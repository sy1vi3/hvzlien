import requests

ENDPOINT = "http://localhost:8080/api/v1"


r = requests.post(ENDPOINT+"/encode/text", json={"text": "i'm a little kitty cat"})

print(r.json())