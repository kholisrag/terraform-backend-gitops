FROM gcr.io/distroless/static-debian11
WORKDIR /go/src/github.com/kholisrag/terraform-backend-gitops

COPY terraform-backend-gitops /usr/local/bin/terraform-backend-gitops
ENTRYPOINT [ "/usr/local/bin/terraform-backend-gitops" ]
