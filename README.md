### Readme is in progress

BucketD is a simple caching service leveraging HTTP and DNS.  

#### Usage Example

Values can be set with an HTTP Post: 

```
curl -X POST 'https://object.patchwork.horse/TestKey?value=TestValue&expire=120' | jq

{
  "key": "testkey",
  "message": "Cache set successfully!",
  "ttl": 120,
  "validUntil": "2026-10-08T18:08:13.670827882Z",
  "warnings": null
}

```

Entries can be retrieved via HTTP using your key: 
```
curl https://object.patchwork.horse/TestKey | jq                          
{
  "ttl": 42,
  "validUntil": "2026-10-08T18:08:13.89297084Z",
  "value": "TestValue"
}
```

Or via a DNS TXT query. Note the TTL used when setting the value is reflected in the DNS response:

```
dig in txt TestKey.object.patchwork.horse +noall +answer
TestKey.object.patchwork.horse. 65 IN   TXT     "TestValue"
```
