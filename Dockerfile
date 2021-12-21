FROM scratch

COPY dsql /dsql

EXPOSE 5480/tcp
EXPOSE 5432/tcp

ENTRYPOINT ["/dsql"]
CMD ["server"]
