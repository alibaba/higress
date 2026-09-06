#!/usr/bin/env python3
# Copyright (c) 2026 Alibaba Group Holding Ltd.
# SPDX-License-Identifier: Apache-2.0
"""Remove only the named A2A test cluster and fixture containers; retain artifacts."""
import os,pathlib,subprocess,sys,json
root=pathlib.Path(sys.argv[1]).resolve()
os.environ['PATH']=str(root/'bin')+':'+os.environ['PATH']
names=['higress-a2a-demo-remote-1','higress-a2a-demo-remote-2']
for name in names:
    probe=subprocess.run(['docker','inspect',name],stdout=subprocess.PIPE,stderr=subprocess.PIPE)
    if probe.returncode:continue
    if json.loads(probe.stdout)[0]['State']['Running']:
        subprocess.run(['docker','kill',name],check=True)
    subprocess.run(['docker','rm',name],check=True)
subprocess.run(['kind','delete','cluster','--name','higress-a2a-demo'],check=True)
containers=subprocess.check_output(['docker','ps','-a','--format','{{.Names}}'],universal_newlines=True).splitlines()
remaining=sorted(set(containers).intersection(names+['higress-a2a-demo-control-plane']))
assert not remaining,remaining
assert 'higress-a2a-demo' not in subprocess.check_output(['kind','get','clusters'],universal_newlines=True).splitlines()
proof={'isolated_cluster_removed':True,'fixture_containers_removed':True,'test_ports_removed_with_cluster':True}
(root/'evidence'/'affinity-cleanup.json').write_text(json.dumps(proof,indent=2))
print(json.dumps(proof))
