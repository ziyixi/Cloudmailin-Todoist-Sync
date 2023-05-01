
from dataclasses import dataclass


@dataclass
class ParsedPost:
    From: str
    To: str
    Date: str
    Subject: str
    Content: str
