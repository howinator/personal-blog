FROM nginx:1.15.8-alpine

COPY nginx.conf /etc/nginx/conf.d/nginx.conf
RUN rm /etc/nginx/conf.d/default.conf

COPY public /usr/share/nginx/html
