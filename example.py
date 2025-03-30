from fastapi import FastAPI
from fastapi.responses import FileResponse
import uvicorn

app = FastAPI()


@app.post("/api/v1/decode")
async def decode():
    return {"phonetic": "ˈkawˌbɔɪz ˈvɪrsəz ˈeɪɪliənz", "letters": "cowboys versus aliens"}

@app.post("/api/v1/encode/text")
async def decode():
    return {"text": "☁☔☃☠☀☆☇☒☤☂☁☡☍☛☜☋☤☂☁☊☒☕☑☋☗☤"}

@app.post("/api/v1/encode/image")
async def decode():
    return FileResponse("./example.png")


