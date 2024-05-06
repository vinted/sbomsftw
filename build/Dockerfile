FROM alpine:latest

ARG TARGETARCH

# We will use this directory for downloads
WORKDIR /opt

# APK requires you to use --no-cache instead of -y and --no-install-recommends
RUN apk --no-cache add curl wget git cmake unzip clang ruby ruby-dev ruby-irb ruby-rake bash ruby-io-console ruby-bigdecimal ca-certificates wget openssl unzip openjdk11

# Node.js and npm setup
RUN apk --no-cache add nodejs npm
RUN npm install -g npm@latest

# Ruby and Gems setup
RUN gem install bundler

# Install yarn & SBOM collection tools (cdxgen & cyclonedx-cli & cyclonedx-tools & licensee)
RUN npm install -g @appthreat/cdxgen retire yarn bower && gem install bundler bundler:1.9 bundler:1.17.3

# Install golang
WORKDIR /opt
RUN wget https://go.dev/dl/go1.21.1.linux-${TARGETARCH}.tar.gz \
  && tar -C /usr/local -xzf /opt/go1.21.1.linux-${TARGETARCH}.tar.gz && rm /opt/go1.21.1.linux-${TARGETARCH}.tar.gz
ENV PATH="/usr/local/go/bin:${PATH}"

# Android SDK setup
ENV ANDROID_HOME=/opt/android-sdk-linux
ENV JAVA_HOME="/usr/lib/jvm/java-11-openjdk"

RUN mkdir -p ${ANDROID_HOME}/cmdline-tools && \
    wget https://dl.google.com/android/repository/commandlinetools-linux-7583922_latest.zip && \
    unzip commandlinetools-linux-7583922_latest.zip -d ${ANDROID_HOME}/cmdline-tools && \
    mv ${ANDROID_HOME}/cmdline-tools/cmdline-tools ${ANDROID_HOME}/cmdline-tools/latest

# Accept Android SDK licenses
RUN yes | ${ANDROID_HOME}/cmdline-tools/latest/bin/sdkmanager --licenses
RUN yes | ${ANDROID_HOME}/cmdline-tools/latest/bin/sdkmanager --update

# This command is not yet run
# Download the necessary Android SDK packages
RUN ${ANDROID_HOME}/cmdline-tools/latest/bin/sdkmanager \
    "platforms;android-28" \
    "build-tools;34.0.0" \
    "extras;android;m2repository" \
    "extras;google;m2repository" \
    "extras;google;google_play_services"

# Setup app
RUN mkdir /builder

# Switch working environment
WORKDIR /builder

# Add application
COPY . .

RUN build/build.sh && mv /builder/sa-collector /usr/local/bin && go clean -modcache && rm -rf /builder
