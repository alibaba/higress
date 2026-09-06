#!/usr/bin/env python3
# Copyright (c) 2026 Alibaba Group Holding Ltd.
# SPDX-License-Identifier: Apache-2.0
"""Render isolated demo resources; no credentials or cluster state are embedded."""
import argparse
import json
p = argparse.ArgumentParser()
p.add_argument('--remote-ip', required=True)
p.add_argument('--remote-endpoints')
p.add_argument('--artifact-path', default='/a2a')
p.add_argument('--variant', choices=['baseline', 'fixed', 'affinity'], default='baseline')
p.add_argument('--affinity', action='store_true')
p.add_argument('--generated', action='store_true', help='Use the new Ingress annotation adapter')
a = p.parse_args()
ns = 'a2a-demo'
resources = [{'apiVersion':'v1','kind':'Namespace','metadata':{'name':ns}}]
image = 'higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/gateway:v2.2.4'
resources += [{'apiVersion':'apps/v1','kind':'Deployment','metadata':{'name':'agent','namespace':ns},'spec':{'replicas':3 if a.affinity else 1,'selector':{'matchLabels':{'app':'a2a-agent'}},'template':{'metadata':{'labels':{'app':'a2a-agent'}},'spec':{'containers':[{'name':'agent','image':image,'command':['/a2a/demo-agent-affinity' if a.affinity else '/a2a/demo-agent'],'args':['-identity=kubernetes-agent'],'ports':[{'containerPort':8080}],'volumeMounts':[{'name':'demo','mountPath':'/a2a','readOnly':True}]}],'volumes':[{'name':'demo','hostPath':{'path':a.artifact_path,'type':'Directory'}}]}}}},
{'apiVersion':'v1','kind':'Service','metadata':{'name':'agent','namespace':ns},'spec':{'selector':{'app':'a2a-agent'},'ports':[{'port':8080,'targetPort':8080}]}}]
resources += [{'apiVersion':'networking.higress.io/v1','kind':'McpBridge','metadata':{'name':'default','namespace':'higress-system'},'spec':{'registries':[{'name':'a2a-remote','type':'static','domain':a.remote_endpoints or a.remote_ip+':8080','port':80}]}}]
if a.affinity:
    resources[-1]['spec']['registries'].append({'name':'a2a-redis','type':'dns','domain':'a2a-redis.a2a-demo.svc.cluster.local','port':6379})
    resources += [
        {'apiVersion':'apps/v1','kind':'Deployment','metadata':{'name':'a2a-redis','namespace':ns},'spec':{'replicas':1,'selector':{'matchLabels':{'app':'a2a-redis'}},'template':{'metadata':{'labels':{'app':'a2a-redis'}},'spec':{'containers':[{'name':'redis','image':'docker.io/tairmodule/tairhash:latest','imagePullPolicy':'Never','command':['redis-server'],'args':['--save','','--appendonly','no'],'ports':[{'containerPort':6379}]}]}}}},
        {'apiVersion':'v1','kind':'Service','metadata':{'name':'a2a-redis','namespace':ns},'spec':{'selector':{'app':'a2a-redis'},'ports':[{'name':'tcp-redis','port':6379}]}}
    ]
for source in ['k8s','remote']:
    host = source+'.agents.test'
    config = {'protocolVersion':'1.0','legacy03':{'enabled':True},'mode':'enforce','agent':{'id':source+'-agent','externalBaseURL':'https://'+host+':30443/'}}
    if a.affinity: config['affinity'] = {'enabled':True,'ttlSeconds':3600,'redis':{'serviceFQDN':'a2a-redis.dns','servicePort':6379,'timeout':1000}}
    annotations = {"higress.io/backend-protocol":"HTTP2"}
    if source == 'remote': annotations['higress.io/destination'] = 'a2a-remote.static:80'
    if a.generated:
        annotations.update({'higress.io/a2a-config':json.dumps(config),'higress.io/a2a-plugin-url':'file:///a2a/'+a.variant+'.wasm'})
    resources.append({'apiVersion':'networking.k8s.io/v1','kind':'Ingress','metadata':{'name':source+'-agent','namespace':ns,'annotations':annotations},'spec':{'ingressClassName':'higress','tls':[{'hosts':[host],'secretName':'demo-tls'}],'rules':[{'host':host,'http':{'paths':[{'path':'/','pathType':'Prefix','backend':{'service':{'name':'agent','port':{'number':8080}}}}]}}]}})
    if source == 'remote':
        resources[-1]['spec']['rules'][0]['http']['paths'][0]['backend'] = {'resource':{'apiGroup':'networking.higress.io','kind':'McpBridge','name':'default'}}
    if not a.generated:
        resources.append({'apiVersion':'extensions.higress.io/v1alpha1','kind':'WasmPlugin','metadata':{'name':source+'-a2a','namespace':'higress-system'},'spec':{'url':'file:///a2a/'+a.variant+'.wasm','phase':'AUTHN','priority':300,'defaultConfigDisable':True,'matchRules':[{'ingress':[ns+'/'+source+'-agent'],'config':config}]}})
print(json.dumps({'apiVersion':'v1','kind':'List','items':resources}, indent=2))
