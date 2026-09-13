FROM node:22-alpine AS frontend
WORKDIR /portal
COPY apps/saas-admin-portal/package.json apps/saas-admin-portal/package-lock.json ./
RUN npm ci
COPY apps/saas-admin-portal/ ./
RUN npm run build

FROM mcr.microsoft.com/dotnet/sdk:8.0 AS build
WORKDIR /src
COPY apps/saas-admin-portal/bff-saas-app/Bff.Saas.App.csproj bff/
RUN dotnet restore bff/Bff.Saas.App.csproj
COPY apps/saas-admin-portal/bff-saas-app/ bff/
RUN dotnet publish bff/Bff.Saas.App.csproj -c Release -o /out

FROM mcr.microsoft.com/dotnet/aspnet:8.0
WORKDIR /app
COPY --from=build /out ./
COPY --from=frontend /portal/dist ./wwwroot
EXPOSE 8080
ENV ASPNETCORE_URLS=http://+:8080
ENTRYPOINT ["dotnet", "Bff.Saas.App.dll"]
