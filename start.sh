#!/bin/sh

/usr/sbin/nginx

while :
do
# run as restricted user
   su -s /usr/local/bin/radicale radicale &
   su -s /caldavserver radicale &
   sleep 1m
# regular cleanup and restart
   echo 'restarting'
   su -s /bin/sh -c "pkill -v start.sh" radicale
   rm -r /var/lib/radicale/collections/*
   mkdir -p /var/lib/radicale/collections/collection-root/jrarj/default
   echo 'BEGIN:VCALENDAR
BEGIN:VEVENT
UID:1
DTEND;TZID="Singapore Standard Time":20220529T094500
DTSTART;TZID="Singapore Standard Time":20220530T091500
SUMMARY:Test Event
END:VEVENT
END:VCALENDAR' > /var/lib/radicale/collections/collection-root/jrarj/default/test.ics
   echo '{"tag": "VCALENDAR"}' > /var/lib/radicale/collections/collection-root/jrarj/default/.Radicale.props
   chown -R radicale:radicale /var/lib/radicale/collections
done

wait -n
exit $?
