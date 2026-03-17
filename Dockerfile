FROM alpine:3 AS fetch
ARG VERSION=latest
RUN apk add --no-cache curl jq \
 && if [ "$VERSION" = "latest" ]; then \
      VERSION=$(curl -fsSL https://api.github.com/repos/samn/gke-cost-analyzer/releases/latest | jq -r .tag_name); \
    fi \
 && curl -fsSL \
      "https://github.com/samn/gke-cost-analyzer/releases/download/${VERSION}/gke-cost-analyzer-linux-amd64.gz" \
    | gunzip > /gke-cost-analyzer \
 && chmod +x /gke-cost-analyzer

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=fetch /gke-cost-analyzer /usr/local/bin/gke-cost-analyzer
ENTRYPOINT ["gke-cost-analyzer"]
