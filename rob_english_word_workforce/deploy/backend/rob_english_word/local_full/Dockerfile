FROM eclipse-temurin:21-jdk-alpine AS build
WORKDIR /app
COPY rob_english_word_back/pom.xml .
COPY rob_english_word_back/src ./src
RUN apk add --no-cache maven && mvn package -DskipTests

FROM eclipse-temurin:21-jre-alpine
WORKDIR /app
COPY --from=build /app/target/*.jar app.jar
EXPOSE 6012 6013
ENTRYPOINT ["java", "-jar", "app.jar"]
