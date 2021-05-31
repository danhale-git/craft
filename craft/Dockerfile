# docker run -d -it -e EULA=TRUE -p 19132:19132/udp danhaledocker/craftmine:latest

FROM ubuntu

# https://www.minecraft.net/en-us/download/server/bedrock > UBUNTU SERVER > DOWNLOAD
# example: https://minecraft.azureedge.net/bin-linux/bedrock-server-1.16.221.01.zip
ARG URL

RUN mkdir bedrock; cd bedrock; \
    echo 'debconf debconf/frontend select Noninteractive' | debconf-set-selections; \
    apt-get update; \
    apt-get install -y wget; \
    apt-get install -y unzip; \
    apt-get install -y libcurl4

RUN cd bedrock; \
    wget $URL; \
    unzip $(echo $URL  | cut -d "/" -f5); \
    chmod +x bedrock_server

RUN mkdir bedrock/worlds; \
    mkdir bedrock/worlds/'Bedrock level' \
    mkdir bedrock/worlds/'Bedrock level'/db
