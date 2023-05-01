# Use the official Python image as the base image
FROM python:3.9-slim as builder

# Set the working directory
WORKDIR /app

# Install curl
RUN apt-get update && \
    apt-get install -y curl && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Copy the 'pyproject.toml' and 'poetry.lock' files into the container
COPY pyproject.toml poetry.lock ./

# Set the PATH environment variable to include Poetry's bin directory
ENV PATH="/root/.local/bin:$PATH"

# Install Poetry and the necessary dependencies
RUN curl -sSL https://install.python-poetry.org | python3 - && \
    poetry config virtualenvs.create false && \
    poetry config installer.parallel true

# Install poetry dependencies
RUN poetry install --no-interaction --no-ansi

# =======================================================================================
# Use the Alpine-based Python image for the final stage
FROM python:3.9-alpine
LABEL org.opencontainers.image.authors="docker@ziyixi.science"
LABEL org.opencontainers.image.source=https://github.com/ziyixi/Cloudmailin-Todoist-Sync
LABEL org.opencontainers.image.description="Python package to handle cloudmailin's post on receiving emails, and update todoist tasks accordingly with ChatGPT's summary"
LABEL org.opencontainers.image.licenses=MIT

# Set the working directory
WORKDIR /app


# Copy the necessary files
COPY --from=builder /usr/local/lib/python3.9/site-packages /usr/local/lib/python3.9/site-packages
COPY src src

ENV PORT=8080

# start the server
CMD ["python", "-m", "src.main"]