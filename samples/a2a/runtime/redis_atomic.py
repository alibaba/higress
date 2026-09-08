#!/usr/bin/env python3
# Copyright (c) 2026 Alibaba Group Holding Ltd.
# SPDX-License-Identifier: Apache-2.0
"""Run the plugin's exact Lua scripts against the isolated demo Redis."""
import concurrent.futures, hashlib, json, os, pathlib, re, subprocess, sys
root=pathlib.Path(sys.argv[1]);os.environ['KUBECONFIG']=str(root/'kubeconfig')
source=(root/'higress/plugins/wasm-go/extensions/a2a-protocol/affinity.go').read_text()
bind=re.search(r'const affinityBind = `([^`]+)`',source).group(1)
lookup=re.search(r'const affinityLookup = `([^`]+)`',source).group(1)
def redis(*args):
    return subprocess.check_output(['kubectl','-n','a2a-demo','exec','deployment/a2a-redis','--','redis-cli','--raw']+list(args),universal_newlines=True).strip()
keys=['a2a:{atomic-verification}:existing','a2a:{atomic-verification}:new']
try:
    redis('DEL',*keys)
    assert redis('EVAL',bind,'1',keys[0],'host-a','60')=='OK'
    conflict=redis('EVAL',bind,'2',keys[1],keys[0],'host-b','60')
    assert 'A2A_BINDING_CONFLICT' in conflict,conflict
    assert redis('GET',keys[0])=='host-a'
    assert redis('EXISTS',keys[1])=='0','Partial write occurred before detecting conflict'
    assert 'A2A_BINDING_MISSING' in redis('EVAL',lookup,'2',*keys,'60')
    redis('DEL',*keys)
    def race(i):return redis('EVAL',bind,'2',*keys,'host-'+str(i%2),'60')
    with concurrent.futures.ThreadPoolExecutor(max_workers=8) as executor:results=list(executor.map(race,range(16)))
    values=[redis('GET',key) for key in keys]
    assert values[0]==values[1] and values[0] in ['host-0','host-1'],values
    assert results.count('OK')==8,results
    assert sum('A2A_BINDING_CONFLICT' in r for r in results)==8,results
    result={'case':'redis_atomic_alias_binding','pass':True,'partial_writes':False,'concurrent_winners':8,'conflicting_writers_rejected':8,'bind_script_sha256':hashlib.sha256(bind.encode()).hexdigest(),'lookup_script_sha256':hashlib.sha256(lookup.encode()).hexdigest()}
    (root/'evidence'/'affinity-redis-atomic.json').write_text(json.dumps(result,indent=2))
    print(json.dumps(result))
finally:redis('DEL',*keys)
