# Copyright (c) 2026 Alibaba Group Holding Ltd.
# SPDX-License-Identifier: Apache-2.0
"""Exercise official A2A SDK discovery, unary, task operations and streaming."""
import asyncio
import json
import sys
import time
import uuid
import httpx
from a2a.client import A2ACardResolver, A2AClient
from a2a.types import (MessageSendParams, SendMessageRequest, SendStreamingMessageRequest,
                      GetTaskRequest, CancelTaskRequest, TaskQueryParams, TaskIdParams)

class KindTransport(httpx.AsyncHTTPTransport):
    async def handle_async_request(self, request):
        # Test-only DNS override. Preserve the gateway Host while dialing kind.
        request.extensions["sni_hostname"] = request.url.host
        request.url = request.url.copy_with(host=sys.argv[1])
        return await super().handle_async_request(request)

async def main():
    async with httpx.AsyncClient(transport=KindTransport(verify=False, http2=True), timeout=10) as http:
        for source in ['k8s', 'remote']:
            base = 'https://' + source + '.agents.test:30443'
            card = await A2ACardResolver(httpx_client=http, base_url=base).get_agent_card()
            assert card.url.rstrip('/') == base, card.url
            expected = 'kubernetes-agent' if source == 'k8s' else 'remote-agent'
            assert card.name == expected, card.name
            client = A2AClient(httpx_client=http, agent_card=card)
            params = MessageSendParams(message={'messageId':str(uuid.uuid4()),'role':'user','parts':[{'kind':'text','text':'hello'}]})
            result = await client.send_message(SendMessageRequest(id='sdk-send', params=params))
            task = result.root.result
            assert task.status.state.value == 'completed'
            assert task.artifacts[0].parts[0].root.text == 'hello from ' + expected
            got = await client.get_task(GetTaskRequest(id='sdk-get', params=TaskQueryParams(id=task.id)))
            assert got.root.result.id == task.id
            # Create a nonterminal task before testing cancellation.
            working = A2AClient(httpx_client=http, url=card.url.rstrip('/')+'/?working=true')
            pending = await working.send_message(SendMessageRequest(id='sdk-working', params=params))
            assert pending.root.result.status.state.value == 'working'
            canceled = await client.cancel_task(CancelTaskRequest(id='sdk-cancel', params=TaskIdParams(id=pending.root.result.id)))
            assert canceled.root.result.status.state.value == 'canceled'
            start = time.monotonic()
            events = []
            async for event in client.send_message_streaming(SendStreamingMessageRequest(id='sdk-stream', params=params)):
                events.append((time.monotonic()-start,event.model_dump(mode='json')))
            assert len(events) == 2, events
            assert events[0][0] < 0.5 and events[1][0] >= 0.6, events
            assert events[-1][1]['result']['final'] is True
            print(json.dumps({'source':source,'sdk':'a2a-sdk==0.3.22','card_url':card.url,'task':task.id,
                              'get':True,'cancel':True,'stream_events':len(events),'first_ms':events[0][0]*1000,'pass':True}))
asyncio.run(main())
