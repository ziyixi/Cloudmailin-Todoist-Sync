import os

import uvicorn
from dotenv import load_dotenv
from fastapi import Depends, FastAPI, HTTPException, Request
from fastapi.security import HTTPBasic, HTTPBasicCredentials

from .handle_post import handle_cmi_post

app = FastAPI()

load_dotenv()  # Load environment variables from .env file if it exists
cmi_user = os.getenv("cloudmailin_username")
cmi_pass = os.getenv("cloudmailin_password")
openai_api_key = os.getenv("openai_api_key")
todoist_api_key = os.getenv("todoist_api_key")

if not cmi_user or not cmi_pass:
    raise Exception(
        "cloudmailin_username or cloudmailin_password is not in .env")

if not openai_api_key:
    raise Exception("openai_api_key is not in .env")

if not todoist_api_key:
    raise Exception("todoist_api_key is not in .env")

security = HTTPBasic()


def auth(credentials: HTTPBasicCredentials = Depends(security)):
    if credentials.username == cmi_user and credentials.password == cmi_pass:
        return True
    raise HTTPException(
        status_code=401, detail="Incorrect username or password")


@app.post("/api/CloudmailinDida365App")
async def protected_handle_cmi_post(request: Request, authorized: bool = Depends(auth)):
    return await handle_cmi_post(request, openai_api_key, todoist_api_key)


def main():
    port = int(os.getenv("PORT", 8000))
    uvicorn.run("src.main:app", host="0.0.0.0", port=port,
                log_level="info", reload=False)


if __name__ == "__main__":
    main()
