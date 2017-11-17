#!/usr/bin/env python
import json
import logging
import pif
import requests
import socket

gandi_host = 'dns.api.gandi.net'
endpoint = 'https://'+gandi_host+'/api/v5'

logging.basicConfig(
    format='%(asctime)s %(message)s',
    level=logging.ERROR)
log = logging.getLogger()

def get_uuid(api_key, domain):
        url = endpoint + '/domains/' + domain
        resp = requests.get(url, headers={'X-Api-Key': api_key})
        resp.raise_for_status()
        return resp.json()['zone_uuid']

def should_update(domain, subdomain):
    try:
        domain = "{0}.{1}".format(subdomain, domain)
        log.debug('domain: %s', domain)
        old_ip = socket.gethostbyname(domain)
        log.debug('old_ip: %s', old_ip)

        public_ip = pif.get_public_ip('dyndns.com')
        log.debug('public ip: %s', public_ip)

        if old_ip != public_ip:
            return public_ip
    except:
        log.exception('should_update')
    return None

def update_dns(public_ip, uuid, key, domain, subdomain):
    url = endpoint + '/zones/' + uuid + '/records/' + subdomain + '/A'
    u = requests.put(url,
        json={
            'rrset_values': [public_ip],
        },
        headers={
            'Content-Type': 'application/json',
            'X-Api-Key': key
        }
    )
    log.info(u.status_code)
    u.raise_for_status()

def is_gandi_reachable():
    try:
        socket.getaddrinfo(gandi_host, 0)
    except:
        return False
    return True

def main():
    if not is_gandi_reachable():
        log.ingo('Gandi Unreachable')
        return
    with open('conf.json') as conf_file:
        conf = json.load(conf_file)
    # to avoid the bad ones more vailable in pif.utils.list_checkers()
        uuid = get_uuid(conf['key'], conf['domain'])
    ip = should_update(conf['domain'], conf['subdomain'])
    if ip:
        update_dns(ip, uuid, **conf)

if __name__ == '__main__':
    main()
