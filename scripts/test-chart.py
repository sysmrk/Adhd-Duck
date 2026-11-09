#!/usr/bin/env python3
"""Render deployment variants and verify security-relevant chart contracts."""
import subprocess
import yaml

def render(*args):
    return list(yaml.safe_load_all(subprocess.check_output(
        ['helm', 'template', 'adhdduck', 'helm/adhdduck', '--namespace', 'alternate', *args], text=True)))

def kind(docs, name):
    return next(x for x in docs if x and x['kind'] == name)

docs = render()
assert not any(x and x['kind'] in ('Ingress', 'ServiceMonitor') for x in docs)
assert kind(docs, 'ConfigMap')['data']['CONTROLLER_URL'] == 'http://adhdduck-controller.alternate.svc.cluster.local:8080'
assert kind(docs, 'ConfigMap')['data']['GC_ENABLED'] == 'false'
assert kind(docs, 'NetworkPolicy')['spec']['policyTypes'] == ['Ingress']
for resource in ('Deployment', 'DaemonSet'):
    pod = kind(docs, resource)['spec']['template']['spec']
    volumes = {x['name']: x for x in pod['volumes']}
    assert volumes['peer-tls']['secret']['secretName'] == 'adhdduck-peer-tls'
    assert not volumes['rescue-token']['secret'].get('optional', False)
    for c in pod['containers']:
        assert c['readinessProbe']['httpGet']['scheme'] == 'HTTPS'
        assert c['livenessProbe']['httpGet']['scheme'] == 'HTTPS'
        assert c['securityContext']['allowPrivilegeEscalation'] is False
        assert c['securityContext']['capabilities']['drop'] == ['ALL']
assert kind(docs, 'DaemonSet')['spec']['template']['spec']['nodeSelector']['kubernetes.io/arch'] == 'amd64'
digest = 'sha256:' + 'a'*64
variant = render('--set', 'controller.image.digest='+digest, '--set', 'ingress.enabled=true',
                 '--set', 'ingress.tls[0].secretName=webhook-tls', '--set', 'ingress.tls[0].hosts[0]=duck.example.com')
assert kind(variant, 'Deployment')['spec']['template']['spec']['containers'][0]['image'].endswith('@'+digest)
ingress = kind(variant, 'Ingress')
assert ingress['spec']['tls'][0]['secretName'] == 'webhook-tls'
assert ingress['metadata']['annotations']['nginx.ingress.kubernetes.io/proxy-ssl-verify'] == 'on'
assert not any(x and x['kind'] == 'NetworkPolicy' for x in render('--set', 'networkPolicy.enabled=false'))
print('Chart security contracts passed')
