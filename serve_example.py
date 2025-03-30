import uvicorn

class App:
    ...

app = App()

if __name__ == "__main__":
    uvicorn.run("example:app", host="0.0.0.0", port=8080, log_level="info")