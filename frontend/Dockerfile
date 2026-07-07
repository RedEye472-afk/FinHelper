# FinHelper-Frontend — Production Dockerfile
# Использует ГОТОВЫЙ dist/ (собранный на хосте), без npm ci внутри Docker
FROM nginx:alpine

RUN rm -rf /usr/share/nginx/html/*
COPY dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget -q --spider http://localhost:80/health || exit 1

CMD ["nginx", "-g", "daemon off;"]