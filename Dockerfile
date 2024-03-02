FROM gcr.io/distroless/static-debian11
WORKDIR /opt/terraform-backend-gitops

COPY terraform-backend-gitops /usr/local/bin/terraform-backend-gitops
ENTRYPOINT [ "/usr/local/bin/terraform-backend-gitops" ]
EXPOSE 20002
CMD [ "serve" ]
