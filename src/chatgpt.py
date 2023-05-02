import openai

TEMPLATE = """
I want you to act as a personal email assistant. I will give you the content of an email, and you will provide a summary of that topic in the format of markdown. Your summary should be concise, covering the most important aspect of the topic. Start your summary with an introductory paragraph that gives an overview of the topic (one sentense, and at most 30 words (important here)!!), then bullet points and sub bullet points to make it more organized. It shouldn't include any level of Headings, an example return format from you should be:
""
Summary in at most one sentenses.

Summary with bullet points:
+ [key information 1]
  - [sub information 1]
  - [sub info 2]
+ [key info 2]
+ [key info 3]
+ ...
""

Below is the content of the email:
""
{email_content}
""
"""


def chat_completation(api_key: str, email_content: str):
    openai.api_key = api_key
    completion = openai.ChatCompletion.create(
        model="gpt-3.`5-turbo", messages=[{"role": "user", "content": TEMPLATE.format(email_content=email_content)}], request_timeout=10000)
    res = completion.choices[0].message.content
    return res
