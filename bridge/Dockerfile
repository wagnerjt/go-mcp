ARG litellm_tag=main-v1.65.0-nightly

FROM ghcr.io/berriai/litellm:${litellm_tag} as litellm

WORKDIR /app

COPY config.yaml .

RUN chmod +x ./docker/entrypoint.sh

EXPOSE 4000/tcp

CMD [ "--port", "4000", "--config", "config.yaml" ]