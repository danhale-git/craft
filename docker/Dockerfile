FROM ubuntu

    # enable libcurl install
RUN mkdir bedrock; cd bedrock; \
    echo 'debconf debconf/frontend select Noninteractive' | debconf-set-selections; \
    apt-get update; \
    apt-get install -y wget; \
    apt-get install -y unzip; \
    apt-get install -y libcurl4

# https://www.minecraft.net/en-us/download/server/bedrock
RUN cd bedrock; \
    wget https://minecraft.azureedge.net/bin-linux/bedrock-server-1.16.210.06.zip; \
    unzip bedrock-server-1.16.210.06.zip

RUN mkdir bedrock/worlds; \
    mkdir bedrock/worlds/'Bedrock level' \
    mkdir bedrock/worlds/'Bedrock level'/db

#COPY craft /usr/local/bin

# CMD cd bedrock; LD_LIBRARY_PATH=. ./bedrock_server

# docker run -d -it -e EULA=TRUE -p 19132:19132/udp danhaledocker/craftmine:v1.2a