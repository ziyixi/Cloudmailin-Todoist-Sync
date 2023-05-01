from todoist_api_python.api import TodoistAPI

from .datatype import ParsedPost

CONTENT_TEMPLATE = """
**FROM: {fromuser}**
**DATE: {date}**
**RECEIVED: {received}**
**SUBJECT: {subject}**

========================
{content}
"""


def add_task_to_todoist(parsed_res: ParsedPost, todoist_api_key: str):
    api = TodoistAPI(todoist_api_key)
    task = api.add_task(
        content=f"{parsed_res.Subject} [{parsed_res.From}]",
        description=CONTENT_TEMPLATE.format(
            fromuser=parsed_res.From,
            date=parsed_res.Date,
            received=parsed_res.To,
            subject=parsed_res.Subject,
            content=parsed_res.Content,
        ),
        labels=["email"],
    )
    return task.url
