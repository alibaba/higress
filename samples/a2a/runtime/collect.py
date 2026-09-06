#!/usr/bin/env python3
# Copyright (c) 2026 Alibaba Group Holding Ltd.
# SPDX-License-Identifier: Apache-2.0
"""Collect demo evidence without copying TLS secrets or full Envoy dumps."""
import json,os,pathlib,subprocess,sys,hashlib
root=pathlib.Path(sys.argv[1]);out=root/'evidence';out.mkdir(exist_ok=True)
os.environ['KUBECONFIG']=str(root/'kubeconfig')
def cmd(*args):return subprocess.check_output(args)
def save(name,data):(out/name).write_bytes(data)
for ns,selector,label in [('higress-system','app=higress-gateway','gateway'),('higress-system','app=higress-controller','controller'),('a2a-demo','app=a2a-agent','agent')]:
    pods=json.loads(cmd('kubectl','-n',ns,'get','pods','-l',selector,'-ojson'))
    save('affinity-'+label+'-pods.json',json.dumps(pods,indent=2).encode())
    for pod in pods['items']:
        name=pod['metadata']['name']
        save(name+'.log',cmd('kubectl','-n',ns,'logs',name,'--all-containers=true'))
        if label=='gateway':
            dump=json.loads(cmd('kubectl','-n',ns,'exec',name,'--','curl','-s','localhost:15000/config_dump'))
            checks={'filter_chains':[],'routes':[]}
            def walk(obj):
                if isinstance(obj,dict):
                    if 'http_filters' in obj:checks['filter_chains'].append([x['name'] for x in obj['http_filters']])
                    if obj.get('name') in ['a2a-demo/k8s-agent','a2a-demo/remote-agent'] and 'route' in obj:
                        checks['routes'].append({k:obj[k] for k in ['name','route','typed_per_filter_config'] if k in obj})
                    for value in obj.values():walk(value)
                elif isinstance(obj,list):
                    for value in obj:walk(value)
            walk(dump)
            assert checks['routes']
            for route in checks['routes']:
                cfg=route['typed_per_filter_config']['higress.a2a.stateful_session']
                assert cfg['stateful_session']['strict'] is True
                assert not route['route'].get('retry_policy') or route['route']['retry_policy'].get('num_retries')==0
            for filters in checks['filter_chains']:
                if 'higress.a2a.stateful_session' not in filters:continue
                index=filters.index('higress.a2a.stateful_session')
                assert filters.index('envoy.filters.http.router')>index
                assert any('a2a-' in f for f in filters[:index]),filters
            save(name+'-routing.json',json.dumps(checks,indent=2).encode())
for resource,ns in [('endpointslices','a2a-demo'),('ingresses','a2a-demo'),('mcpbridge','higress-system'),('deployments','a2a-demo'),('deployments','higress-system')]:
    save('affinity-'+ns+'-'+resource+'.json',cmd('kubectl','-n',ns,'get',resource,'-ojson'))
files=['affinity-final.wasm','demo-agent-affinity-v2','higress-controller-affinity-final','redis.tar']
lines=[]
for name in files:
    with open(root/name,'rb') as file:
        h=hashlib.sha256()
        for block in iter(lambda:file.read(1<<20),b''):h.update(block)
    lines.append(h.hexdigest()+'  '+name)
save('affinity-artifacts.sha256',('\n'.join(lines)+'\n').encode())
print('Collected route-order, retry, replica, discovery and artifact evidence')
