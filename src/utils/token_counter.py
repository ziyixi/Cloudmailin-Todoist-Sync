# This is a work in progress. There are still bugs. Once it is production-ready this will become a full repo.
# from https://gist.github.com/buanzo/7cdd2c34fc0bb25c71b857a16853c6fa
import tiktoken


def count_tokens(text, model_name="gpt-3.5-turbo"):
    encoding = tiktoken.encoding_for_model(model_name)
    num_tokens = len(encoding.encode(text))
    return num_tokens
