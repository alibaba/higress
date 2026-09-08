# Copyright (c) 2026 Alibaba Group Holding Ltd.
# SPDX-License-Identifier: Apache-2.0
"""Fresh HTTP/2 connections alternate between two independently addressed gateways."""
import asyncio, json, sys, time, uuid, os
import httpx
NODE = sys.argv[1]
MODE = sys.argv[2] if len(sys.argv)>2 else 'fixed'
PORTS = [30081,30082]
class Transport(httpx.AsyncHTTPTransport):
    def __init__(self, gateway):
        super().__init__(verify=False,http2=True)
        self.port=PORTS[gateway]
    async def handle_async_request(self, request):
        request.extensions['sni_hostname']=request.url.host
        request.url=request.url.copy_with(host=NODE,port=self.port)
        return await super().handle_async_request(request)

def message(context=None,task=None):
    m={'messageId':str(uuid.uuid4()),'role':'user','parts':[{'kind':'text','text':'affinity'}]}
    if context:m['contextId']=context
    if task:m['taskId']=task
    return {'message':m}

async def rpc(source,gw,method,params,path='/?working=true',expected=200):
    request_id=str(uuid.uuid4())
    async with httpx.AsyncClient(transport=Transport(gw),timeout=10) as c:
        r=await c.post('https://'+source+'.agents.test:30443'+path,json={'jsonrpc':'2.0','id':request_id,'method':method,'params':params},headers={} if MODE=='baseline' else {'x-higress-a2a-affinity-endpoint':'MTAuMC4wLjE6ODA4MA==','x-envoy-retry-on':'5xx','x-envoy-max-retries':'10'})
        assert r.status_code==expected,(r.status_code,r.text)
        assert 'x-higress-a2a-affinity-endpoint' not in r.headers or MODE=='baseline'
        d=r.json();assert d['id']==request_id,d
        return d

async def stream(source,gw,method,params):
    start=time.monotonic(); events=[]
    async with httpx.AsyncClient(transport=Transport(gw),timeout=10) as c:
        async with c.stream('POST','https://'+source+'.agents.test:30443/',json={'jsonrpc':'2.0','id':'stream','method':method,'params':params}) as r:
            assert r.status_code==200,r.status_code
            async for line in r.aiter_lines():
                if line.startswith('data: '):
                    event=json.loads(line[6:]);assert 'error' not in event,event
                    events.append((time.monotonic()-start,event['result']))
                    if len(events)==1:
                        task=event['result']
                        # The other gateway must resolve the binding before the
                        # original SSE stream has completed.
                        lookup_start=time.monotonic()
                        got=await rpc(source,1-gw,'tasks/get',{'id':task['id']})
                        assert time.monotonic()-lookup_start<0.3, 'Task lookup blocked behind SSE completion'
                        assert got['result']['metadata']['node']==task['metadata']['node']
    assert len(events)==2,events
    assert events[0][0]<0.5 and events[1][0]>=0.6,events
    return events

async def main():
    output={}
    for source in ['k8s','remote']:
        tasks=[];missing=0
        for i in range(12):
            task=(await rpc(source,i%2,'message/send',message()))['result'];tasks.append(task)
            for gw in [0,1]:
                got=await rpc(source,gw,'tasks/get',{'id':task['id']})
                if 'error' in got:missing+=1
                elif MODE!='baseline':assert got['result']['metadata']['node']==task['metadata']['node']
        if MODE=='baseline':
            assert missing>0,'Expected unbound multi-replica failures'
            print(json.dumps({'source':source,'baseline_lost_task_requests':missing,'pass':True}),flush=True);continue
        assert missing==0,missing
        nodes={t['metadata']['node'] for t in tasks};assert len(nodes)>=2,nodes
        task=tasks[0]
        replies=await asyncio.gather(*[rpc(source,i%2,'message/send',message(context=task['contextId'])) for i in range(10)])
        turns=[]
        for i,reply in enumerate(replies):
            result=reply['result'];assert result['metadata']['node']==task['metadata']['node'];turns.append(result['metadata']['turn'])
            got=await rpc(source,1-i%2,'tasks/get',{'id':result['id']});assert got['result']['id']==result['id']
        assert sorted(turns)==list(range(2,12)),turns
        other=next(t for t in tasks if t['metadata']['node']!=task['metadata']['node'])
        conflict=await rpc(source,1,'message/send',message(context=other['contextId'],task=task['id']),expected=503)
        assert conflict['error']['code']==-32010
        missing=await rpc(source,1,'tasks/get',{'id':'missing-'+str(uuid.uuid4())},expected=503)
        assert missing['error']['code']==-32010
        long_id=await rpc(source,0,'tasks/get',{'id':'x'*300},expected=503)
        assert long_id['error']['code']==-32010
        canceled=await rpc(source,1,'tasks/cancel',{'id':task['id']})
        assert canceled['result']['status']['state']=='canceled'
        events=await stream(source,0,'message/stream',message())
        subscribed=await stream(source,1,'tasks/resubscribe',{'id':tasks[1]['id']})
        assert subscribed[0][1]['metadata']['node']==tasks[1]['metadata']['node']
        # A task learned on one Agent route must not bind on another route.
        cross=await rpc('remote' if source=='k8s' else 'k8s',0,'tasks/get',{'id':task['id']},expected=503)
        assert cross['error']['code']==-32010
        output[source]=tasks
        print(json.dumps({'source':source,'gateways':2,'agent_nodes':sorted(nodes),'task_get_checks':34,'concurrent_context_turns':turns,'conflict_rejected':True,'unknown_rejected':True,'cross_route_rejected':True,'cancel':True,'stream':True,'resubscribe':True,'stream_first_ms':events[0][0]*1000,'pass':True}),flush=True)
    if MODE!='baseline':
        with open(os.path.join(os.environ['A2A_ARTIFACT_ROOT'],'evidence','affinity-sessions.json'),'w') as f:json.dump(output,f)
if __name__=='__main__':asyncio.run(main())
