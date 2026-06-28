build из корня monorepo

# Next.js
docker build --build-arg APP=sim-ui --target next -t sim-ui frontend/

docker build --build-arg APP=web --target next -t web frontend/

# Vite
docker build --build-arg APP=apartment-ui --target vite -t apartment-ui frontend/

run

# sim-ui (Next.js, порт 3000)
docker run -p 3000:3000 sim-ui

# web (Next.js, порт 3000)
docker run -p 3000:3000 web

# apartment-ui (Vite, nginx порт 80)
docker run -p 8080:80 apartment-ui