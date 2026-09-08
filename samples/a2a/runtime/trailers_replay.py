#!/usr/bin/env python3
# Copyright (c) 2026 Alibaba Group Holding Ltd.
# SPDX-License-Identifier: Apache-2.0
"""Compare baseline/fixed HTTP trailers on the same single-endpoint fixtures.
Run after the affinity suite: this intentionally changes the demo topology.
The baseline.wasm artifact must be built from the revision linked in RESULTS.md.
"""
import json, os, pathlib, subprocess, sys, time
root=pathlib.Path(sys.argv[1]).resolve();node=sys.argv[2]
os.environ['KUBECONFIG']=str(root/'kubeconfig')
def kubectl(*args):return subprocess.check_output(['kubectl']+list(args),universal_newlines=True)
assert (root/'baseline.wasm').is_file()
kubectl('-n','a2a-demo','scale','deployment/agent','--replicas=1')
kubectl('-n','a2a-demo','rollout','status','deployment/agent','--timeout=120s')
# Select a single currently registered external endpoint, preserving other sources.
bridge=json.loads(kubectl('-n','higress-system','get','mcpbridge','default','-ojson'))
endpoint=bridge['spec']['registries'][0]['domain'].split(',')[0]
kubectl('-n','higress-system','patch','mcpbridge','default','--type=json','-p',json.dumps([{'op':'replace','path':'/spec/registries/0/domain','value':endpoint}]))
for variant,artifact in [('baseline','baseline.wasm'),('fixed','affinity-final.wasm')]:
    for source in ['k8s','remote']:
        ingress=json.loads(kubectl('-n','a2a-demo','get','ingress',source+'-agent','-ojson'))
        config=json.loads(ingress['metadata']['annotations']['higress.io/a2a-config'])
        config.pop('affinity',None)
        kubectl('-n','a2a-demo','annotate','ingress',source+'-agent','higress.io/a2a-config='+json.dumps(config),'higress.io/a2a-plugin-url=file:///a2a/'+artifact,'--overwrite')
    time.sleep(5)
    with (root/'evidence'/('trailers-'+variant+'.jsonl')).open('w') as output:
        subprocess.run(['go','run',str(root/'higress/samples/a2a/runtime/verify/main.go'),'-node',node,'-variant',variant],stdout=output,check=True)
    print(variant+': 34 assertions passed',flush=True)
