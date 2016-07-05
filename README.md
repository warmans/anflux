Anflux
==============

Grafana annotations API for InfluxDB storage.

e.g. to add an annotation:

```
curl -i -X POST -H "Content-Type: text/plain" -d "the message body" http://localhost:8888/note/foo/bar?title=whatever
```

to observe annotations