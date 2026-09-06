#!/usr/bin/env python3
# Copyright (c) 2026 Alibaba Group Holding Ltd.
# SPDX-License-Identifier: Apache-2.0
"""Recreate the isolated kind demo from the retained, locally built artifacts.
Run on the development machine with the artifact directory as argv[1].
Refuses to reuse an existing cluster or remote fixture container.
"""
import json, os, pathlib, subprocess, sys, time
root=pathlib.Path(sys.argv[1]).resolve()
name='higress-a2a-demo'
os.environ['PATH']=str(root/'bin')+':'+os.environ['PATH']
os.environ['KUBECONFIG']=str(root/'kubeconfig')
def run(*cmd,**kw):
    print('+ '+' '.join(str(x) for x in cmd),flush=True)
    return subprocess.run([str(x) for x in cmd],check=True,**kw)
def output(*cmd):return subprocess.check_output([str(x) for x in cmd],universal_newlines=True)
def apply(obj):run('kubectl','apply','-f','-',input=json.dumps(obj).encode())
def patch(namespace,deploy,container,binary=None):
    c={'name':container,'volumeMounts':[{'name':'a2a','mountPath':'/a2a','readOnly':True}]}
    if binary:c['command']=['/a2a/'+binary]
    data={'spec':{'template':{'spec':{'containers':[c],'volumes':[{'name':'a2a','hostPath':{'path':'/a2a','type':'Directory'}}]}}}}
    run('kubectl','-n',namespace,'patch','deployment',deploy,'--type=strategic','-p',json.dumps(data))
for file in ['images.tar','redis.tar','higress-controller-affinity-final','affinity-final.wasm','demo-agent-affinity-v2']:
    assert (root/file).is_file(),file
assert name not in output('kind','get','clusters').splitlines(),'Delete only the previous isolated demo first'
for remote in ['higress-a2a-demo-remote-1','higress-a2a-demo-remote-2']:
    assert subprocess.run(['docker','inspect',remote],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL).returncode!=0,remote+' already exists'
(root/'kind.yaml').write_text('kind: Cluster\napiVersion: kind.x-k8s.io/v1alpha4\nnodes:\n- role: control-plane\n  extraMounts:\n  - hostPath: '+str(root)+'\n    containerPath: /a2a\n')
# --retain permits the documented cgroup-v1 workaround if kubelet bootstrap fails.
process=subprocess.Popen(['kind','create','cluster','--name',name,'--image','docker.io/kindest/node:v1.34.0','--config',str(root/'kind.yaml'),'--kubeconfig',str(root/'kubeconfig'),'--retain','--wait','60s'])
# Create the missing cgroup directory while kubeadm is waiting, not after its
# bootstrap timeout. On cgroup v2 this command is a no-op.
for _ in range(90):
    if process.poll() is not None:break
    fixed=subprocess.run(['docker','exec',name+'-control-plane','sh','-c','if test -d /sys/fs/cgroup/systemd; then mkdir -p /sys/fs/cgroup/systemd/kubelet.slice/kubelet-kubepods.slice && systemctl restart kubelet; fi'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
    if fixed.returncode==0:break
    time.sleep(1)
assert process.wait()==0, 'kind bootstrap failed; retained only the new test node for diagnosis'
run('kind','export','kubeconfig','--name',name,'--kubeconfig',root/'kubeconfig')
run('kubectl','wait','--for=condition=Ready','node','--all','--timeout=120s')
for file in ['images.tar','redis.tar']:run('docker','exec',name+'-control-plane','ctr','-n','k8s.io','images','import','/a2a/'+file)
run('helm','upgrade','--install','higress',root/'higress/helm/core','-n','higress-system','--create-namespace','--set','global.tag=v2.2.4','--set','gateway.replicas=2','--set','gateway.service.type=ClusterIP','--set','global.enableRedis=false')
patch('higress-system','higress-controller','higress-core','higress-controller-affinity-final')
patch('higress-system','higress-gateway','higress-gateway')
endpoints=[]
for remote in ['higress-a2a-demo-remote-1','higress-a2a-demo-remote-2']:
    run('docker','run','-d','--name',remote,'--network','kind','-v',str(root)+':/a2a:ro','--entrypoint','/a2a/demo-agent-affinity-v2','docker.io/kindest/node:v1.34.0','-identity=remote-agent')
    info=json.loads(output('docker','inspect',remote))[0]
    endpoints.append(info['NetworkSettings']['Networks']['kind']['IPAddress']+':8080')
manifest=json.loads(output('python3',root/'higress/samples/a2a/runtime/render.py','--remote-ip',endpoints[0].split(':')[0],'--remote-endpoints',','.join(endpoints),'--variant','affinity','--generated','--affinity'))
for item in manifest['items']:
    if item['kind']=='Ingress':item['metadata']['annotations']['higress.io/a2a-plugin-url']='file:///a2a/affinity-final.wasm'
    if item['kind']=='Deployment' and item['metadata']['name']=='agent':item['spec']['template']['spec']['containers'][0]['command']=['/a2a/demo-agent-affinity-v2']
(root/'affinity.json').write_text(json.dumps(manifest,indent=2))
apply(manifest)
run('openssl','req','-x509','-newkey','rsa:2048','-nodes','-keyout',root/'tls.key','-out',root/'tls.crt','-days','7','-subj','/CN=k8s.agents.test','-addext','subjectAltName=DNS:k8s.agents.test,DNS:remote.agents.test',stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
run('kubectl','-n','a2a-demo','create','secret','tls','demo-tls','--cert='+str(root/'tls.crt'),'--key='+str(root/'tls.key'))
for ns,deploy in [('a2a-demo','agent'),('a2a-demo','a2a-redis'),('higress-system','higress-controller'),('higress-system','higress-gateway')]:
    run('kubectl','-n',ns,'rollout','status','deployment/'+deploy,'--timeout=120s')
pods=json.loads(output('kubectl','-n','higress-system','get','pods','-l','app=higress-gateway','-ojson'))['items']
pods=[p for p in pods if not p['metadata'].get('deletionTimestamp') and all(c.get('ready') for c in p['status'].get('containerStatuses',[]))]
assert len(pods)==2
for i,pod in enumerate(pods):
    run('kubectl','-n','higress-system','label','pod',pod['metadata']['name'],'a2a-test-gateway='+str(i),'--overwrite')
    apply({'apiVersion':'v1','kind':'Service','metadata':{'name':'a2a-gateway-'+str(i),'namespace':'higress-system'},'spec':{'type':'NodePort','selector':{'a2a-test-gateway':str(i)},'ports':[{'port':443,'targetPort':443,'nodePort':30081+i}]}})
apply({'apiVersion':'v1','kind':'Service','metadata':{'name':'a2a-gateway','namespace':'higress-system'},'spec':{'type':'NodePort','selector':{'app':'higress-gateway'},'ports':[{'port':443,'targetPort':443,'nodePort':30443}]}})
print('Demo ready. Node IP: '+json.loads(output('docker','inspect',name+'-control-plane'))[0]['NetworkSettings']['Networks']['kind']['IPAddress'])
