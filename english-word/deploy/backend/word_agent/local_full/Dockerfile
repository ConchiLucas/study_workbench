ARG PROJECT_IMAGE=word-agent:1.0.0
FROM ${PROJECT_IMAGE}

WORKDIR /app

ENV PYTHONUNBUFFERED=1 \
    PATH="/app/.venv/bin:$PATH" \
    PYTHONPATH=/app/src

COPY word_select_dashboard/word-agent/. .

EXPOSE 6017

CMD ["python", "-m", "word_agent.main"]
