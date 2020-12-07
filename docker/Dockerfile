FROM ubuntu

    # enable libcurl install
RUN echo 'debconf debconf/frontend select Noninteractive' | debconf-set-selections; \
    apt-get update; \
    apt-get install -y wget; \
    apt-get install -y unzip; \
    apt-get install -y libcurl4

RUN wget https://minecraft.azureedge.net/bin-linux/bedrock-server-1.16.100.04.zip; \
    unzip bedrock-server-1.16.100.04.zip;

COPY craft /usr/local/bin


#CMD LD_LIBRARY_PATH=. ./bedrock_server

# docker run -d -it -e EULA=TRUE -p 19132:19132/udp danhaledocker/craftmine:v1.2a