# Copyright (c) 2026 Alibaba Group Holding Ltd.
# SPDX-License-Identifier: Apache-2.0
"""Destructive fault tests limited to this demo's namespace and containers."""
import asyncio, hashlib, json, os, subprocess, time
from affinity_verify import rpc, message, Transport
import httpx
ROOT=os.environ['A2A_ARTIFACT_ROOT']
os.environ['KUBECONFIG']=ROOT+'/kubeconfig'
def kubectl(*args):return subprocess.check_output(['kubectl',*args],text=True)
def redis(*args):return kubectl('-n','a2a-demo','exec','deployment/a2a-redis','--','redis-cli','--raw',*args).strip()
def record(case,**extra):print(json.dumps({'case':case,'pass':True,**extra}),flush=True)
async def wait_for(predicate):
    for _ in range(30):
        if predicate():return
        await asyncio.sleep(1)
    raise AssertionError('Discovery did not converge')
async def must_fail(source,task):
    for gw in [0,1]:
        d=await rpc(source,gw,'tasks/get',{'id':task['id']},expected=503)
        assert d['error']['code']==-32010,d
async def main():
    sessions=json.load(open(ROOT+'/evidence/affinity-sessions.json'))
    task=sessions['k8s'][1]
    pod=task['metadata']['node'].rsplit('-',1)[0]
    # Continue traffic while removing the selected Pod. A success must still
    # identify the original process; no successful fallback or TaskNotFound.
    counts={'original_success':0,'unavailable':0}
    async def traffic():
        for i in range(30):
            async with httpx.AsyncClient(transport=Transport(i%2),timeout=15) as client:
                r=await client.post('https://k8s.agents.test:30443/',json={'jsonrpc':'2.0','id':'during-delete','method':'tasks/get','params':{'id':task['id']}})
                if r.status_code==503:counts['unavailable']+=1
                else:
                    assert r.status_code==200,(r.status_code,r.text)
                    assert r.json()['result']['metadata']['node']==task['metadata']['node'],r.text
                    counts['original_success']+=1
            await asyncio.sleep(.05)
    traffic_task=asyncio.create_task(traffic())
    await asyncio.sleep(.2)
    kubectl('-n','a2a-demo','delete','pod',pod,'--wait=false')
    await traffic_task
    await must_fail('k8s',task)
    assert counts['unavailable']>0
    fresh=(await rpc('k8s',0,'message/send',message()))['result']
    assert fresh['metadata']['node']!=task['metadata']['node']
    record('pod_removal_during_traffic',deleted=pod,**counts)

    remote=sessions['remote'][0]
    prefix=remote['metadata']['node'].split('-')[0]
    infos=json.loads(subprocess.check_output(['docker','inspect','higress-a2a-demo-remote-1','higress-a2a-demo-remote-2']))
    dead=next(x for x in infos if x['Id'].startswith(prefix))
    live=next(x for x in infos if x['Id']!=dead['Id'])
    subprocess.check_call(['docker','kill',dead['Name'].lstrip('/')],stdout=subprocess.DEVNULL)
    # EDS still advertises the stopped process: strict routing + disabled retries
    # must return unavailable, despite a second live Agent endpoint.
    for gw in [0,1]:
        async with httpx.AsyncClient(transport=Transport(gw),timeout=15) as client:
            r=await client.post('https://remote.agents.test:30443/',json={'jsonrpc':'2.0','id':'stopped','method':'tasks/get','params':{'id':remote['id']}})
            assert r.status_code==503,(r.status_code,r.text)
    endpoint=live['NetworkSettings']['Networks']['kind']['IPAddress']+':8080'
    kubectl('-n','higress-system','patch','mcpbridge','default','--type=json','-p',json.dumps([{'op':'replace','path':'/spec/registries/0/domain','value':endpoint}]))
    await asyncio.sleep(2)
    await must_fail('remote',remote)
    live_task=(await rpc('remote',1,'message/send',message()))['result']
    assert live_task['metadata']['node']!=remote['metadata']['node']
    record('remote_stop_and_source_removal',removed=dead['Name'],remaining=endpoint)

    task=(await rpc('k8s',0,'message/send',message()))['result']
    keys=redis('--scan','--pattern','a2a:*').splitlines()
    suffixes=[hashlib.sha256(task[field].encode()).hexdigest() for field in ['id','contextId']]
    aliases=[key for key in keys if any(key.endswith(s) for s in suffixes)]
    assert len(aliases)==2,aliases
    for key in aliases:redis('PEXPIRE',key,'1000')
    await asyncio.sleep(1.2)
    await must_fail('k8s',task)
    assert all(redis('EXISTS',key)=='0' for key in aliases)
    record('expired_bindings_are_not_reassigned')

    stream_events=[]
    async with httpx.AsyncClient(transport=Transport(0),timeout=15) as client:
        async with client.stream('POST','https://k8s.agents.test:30443/',json={'jsonrpc':'2.0','id':'stream-store-outage','method':'message/stream','params':message()}) as response:
            async for line in response.aiter_lines():
                if not line.startswith('data: '):continue
                stream_events.append(json.loads(line[6:]))
                if len(stream_events)==1:
                    assert 'result' in stream_events[0]
                    kubectl('-n','a2a-demo','scale','deployment/a2a-redis','--replicas=0')
    assert stream_events[-1].get('error',{}).get('code')==-32010,stream_events
    assert not any(e.get('result',{}).get('final') for e in stream_events),stream_events
    record('sse_store_failure_terminates_with_error',events=len(stream_events))
    await wait_for(lambda:not json.loads(kubectl('-n','a2a-demo','get','endpoints','a2a-redis','-ojson')).get('subsets'))
    await must_fail('k8s',fresh)
    kubectl('-n','a2a-demo','scale','deployment/a2a-redis','--replicas=1')
    # Redis in this failure fixture intentionally has no persistence. Lost
    # mappings remain unavailable after recovery; new sessions become usable.
    await wait_for(lambda:bool(json.loads(kubectl('-n','a2a-demo','get','endpoints','a2a-redis','-ojson')).get('subsets')))
    for _ in range(10):
        try:
            new=(await rpc('k8s',0,'message/send',message()))['result'];break
        except AssertionError:await asyncio.sleep(1)
    else:raise AssertionError('Redis recovery failed')
    await must_fail('k8s',fresh)
    got=await rpc('k8s',1,'tasks/get',{'id':new['id']})
    assert got['result']['id']==new['id']
    record('redis_outage_and_binding_loss_fail_closed')
if __name__=='__main__':asyncio.run(main())
