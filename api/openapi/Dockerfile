FROM swaggerapi/swagger-ui:latest

COPY *.yml /usr/share/nginx/html/

# Don't attempt spec validation at swagger.io
#ENV VALIDATOR_URL ""

ENV URLS '[{url:"swagger.yml",name:"cosmos-sdk"}, {url:"clp_get.yml", name:"clp_get"}, {url:"clp_post.yml", name: "clp_post"}]'

RUN chmod 644 /usr/share/nginx/html/swagger.yml

# Disable all caching in Nginx so edits to Swagger aren't cached on browser.
RUN sed -i 's/expires 1d/expires -1/' /etc/nginx/nginx.conf

EXPOSE 8080

CMD ["sh", "/usr/share/nginx/run.sh"]
