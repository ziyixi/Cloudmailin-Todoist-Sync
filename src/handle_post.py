import json
import re

from fastapi import HTTPException, Request
from markdownify import markdownify as md
from tenacity import RetryCallState, retry, stop_after_attempt, wait_fixed

from .chatgpt import chat_completation
from .datatype import ParsedPost
from .todoist import add_task_to_todoist


def retry_if_exception(result: RetryCallState):
    return result.outcome.failed


@retry(retry=retry_if_exception, stop=stop_after_attempt(5), wait=wait_fixed(1))
async def chat_completation_with_retry(api_key: str, email_content: str):
    return chat_completation(api_key, email_content)


def parse_json(s: str) -> ParsedPost:
    data = json.loads(s)
    html = data.get("html", "")

    markdown_raw = md(html, strip=['a'])
    if not markdown_raw:
        markdown_raw = data.get("plain", "")

    markdown = re.sub(r'\[([^\]]+)\]\(([^)]+)\)', r'[\1]()', markdown_raw)

    headers = data.get("headers", {})
    envelope = data.get("envelope", {})

    res = ParsedPost(
        From=headers.get("from", ""),
        To=headers.get("to", ""),
        Date=headers.get("date", ""),
        Subject=headers.get("subject", ""),
        Content=markdown,
    )

    helo_domain = envelope.get("helo_domain", "")
    if "outlook" in helo_domain and res.Subject.startswith("FW: "):
        res.Subject = res.Subject[4:]

    if "cloudmailin" in res.To:
        re_pattern = r'_+\\r\\nFrom: .*?([a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+)'
        re_compiled = re.compile(re_pattern)
        matches = re_compiled.findall(s)
        if len(matches) < 1:
            res.To = res.From
            res.From = "sender unknown"
        else:
            res.To = res.From
            res.From = matches[0]

    return res


async def handle_cmi_post(request: Request, openai_api_key: str, todoist_api_key: str):
    data_raw = await request.body()
    data = data_raw.decode("utf-8")
    parsed_res = parse_json(data)
    if not parsed_res.From:
        raise HTTPException(status_code=400, detail="From is empty")
    if len(parsed_res.Content) > 0:
        try:
            parsed_res.Content = await chat_completation_with_retry(openai_api_key, parsed_res.Content)
        except Exception as e:
            raise HTTPException(status_code=500, detail=str(e))
    else:
        parsed_res.Content = "No content"

    # add task to todoist inbox
    try:
        url = add_task_to_todoist(parsed_res, todoist_api_key)
        return {"url": url}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
