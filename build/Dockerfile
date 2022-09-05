FROM debian:bullseye-slim

ENV DEBIAN_FRONTEND noninteractive
ENV NODE_KEY="/usr/share/keyrings/nodesource.gpg"

# Install core apt dependencies
RUN apt-get update && apt-get install apt-utils software-properties-common gnupg ca-certificates \
    curl wget -y --no-install-recommends

# Import third-party PGP keys
RUN wget -qO - https://adoptopenjdk.jfrog.io/adoptopenjdk/api/gpg/key/public | apt-key add - \
    && curl -s https://deb.nodesource.com/gpgkey/nodesource.gpg.key | gpg --dearmor | tee ${NODE_KEY} >/dev/null \
    && curl -sL "https://keyserver.ubuntu.com/pks/lookup?op=get&search=0x2EE0EA64E40A89B84B2DF73499E82A75642AC823" \
    | gpg --no-default-keyring --keyring gnupg-ring:/etc/apt/trusted.gpg.d/scalasbt-release.gpg --import \
    && chmod 644 /etc/apt/trusted.gpg.d/scalasbt-release.gpg

# Setup third-party apt repositories
WORKDIR /etc/apt/sources.list.d
RUN add-apt-repository --yes https://adoptopenjdk.jfrog.io/adoptopenjdk/deb/ \
    && echo "deb [signed-by=${NODE_KEY}] https://deb.nodesource.com/node_12.x bullseye main" > nodesource.list \
    && echo "deb-src [signed-by=${NODE_KEY}] https://deb.nodesource.com/node_12.x bullseye main" >> nodesource.list \
    && echo "deb https://repo.scala-sbt.org/scalasbt/debian all main" | tee /etc/apt/sources.list.d/sbt.list \
    && echo "deb https://repo.scala-sbt.org/scalasbt/debian /" | tee /etc/apt/sources.list.d/sbt_old.list \

# Install all remaining apt dependencies
RUN apt-get update && apt-get install rubygems build-essential git ruby-dev adoptopenjdk-8-hotspot \
    cmake maven pkg-config libssl-dev locales unzip clang nodejs -y --no-install-recommends

# Fix up locales
RUN sed -i '/en_US.UTF-8/s/^# //g' /etc/locale.gen && locale-gen
ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US:en
ENV LC_ALL=en_US.UTF-8

# Install yarn & SBOM collection tools (cdxgen & cyclonedx-cli & cyclonedx-tools & licensee)
RUN npm install -g @appthreat/cdxgen retire yarn bower && gem install bundler bundler:1.9 bundler:1.17.3

# Install golang
WORKDIR /opt
RUN wget https://go.dev/dl/go1.18.1.linux-amd64.tar.gz \
  && tar -C /usr/local -xzf /opt/go1.18.1.linux-amd64.tar.gz && rm /opt/go1.18.1.linux-amd64.tar.gz
ENV PATH="/usr/local/go/bin:${PATH}"

#Install Android SDK & NDK
### Android environment variables
ENV ANDROID_HOME="/root/android-sdk-linux"
ENV PATH="${PATH}:${ANDROID_HOME}/tools/bin:${ANDROID_HOME}/platform-tools:${ANDROID_HOME}/tools"
ENV ANDROID_BUILD_TOOLS_VERSION="4333796"

### Setup Android SDK using Java 8 & Accept EULA
ENV JAVA_HOME="/usr/lib/jvm/adoptopenjdk-8-hotspot-amd64"
RUN  mkdir -p ~/.android && touch ~/.android/repositories.cfg && mkdir ~/android-sdk-linux && cd ~/android-sdk-linux \
    && wget https://dl.google.com/android/repository/sdk-tools-linux-${ANDROID_BUILD_TOOLS_VERSION}.zip \
    -q --output-document=sdk-tools.zip && unzip sdk-tools.zip && rm sdk-tools.zip \
    && echo y | sdkmanager "build-tools;28.0.2" "platforms;android-28" \
    && echo y | sdkmanager "extras;android;m2repository" "extras;google;m2repository" "extras;google;google_play_services"

# Setup app
RUN mkdir /builder

# Switch working environment
WORKDIR /builder

# Add application
COPY . .

RUN build/build.sh && mv /builder/sa-collector /usr/local/bin && go clean -modcache && rm -rf /builder