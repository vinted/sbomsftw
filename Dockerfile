FROM golang:1.18

ENV DEBIAN_FRONTEND noninteractive
ENV NODE_KEY="/usr/share/keyrings/nodesource.gpg"

# Install core apt dependencies
RUN apt-get update && apt-get install apt-utils software-properties-common gnupg ca-certificates \
    curl wget -y --no-install-recommends

# Import third-party PGP keys
RUN wget -qO - https://adoptopenjdk.jfrog.io/adoptopenjdk/api/gpg/key/public | apt-key add - \
    && curl -s https://deb.nodesource.com/gpgkey/nodesource.gpg.key | gpg --dearmor | tee ${NODE_KEY} >/dev/null

# Setup third-party apt repositories
WORKDIR /etc/apt/sources.list.d
RUN add-apt-repository --yes https://adoptopenjdk.jfrog.io/adoptopenjdk/deb/ \
    && echo "deb [signed-by=${NODE_KEY}] https://deb.nodesource.com/node_12.x focal main" > nodesource.list \
    && echo "deb-src [signed-by=${NODE_KEY}] https://deb.nodesource.com/node_12.x focal main" >> nodesource.list

# Install all remaining apt dependencies
RUN apt-get update && apt-get install rubygems build-essential git ruby-dev adoptopenjdk-8-hotspot \
    cmake maven pkg-config libssl-dev locales unzip clang nodejs -y --no-install-recommends

# Fix up locales
RUN sed -i '/en_US.UTF-8/s/^# //g' /etc/locale.gen && locale-gen
ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US:en
ENV LC_ALL=en_US.UTF-8

# Install yarn & SBOM collection tools (cdxgen & cyclonedx-cli & cyclonedx-tools & licensee)
RUN npm install -g @appthreat/cdxgen yarn bower && gem install bundler bundler:1.9 bundler:1.17.3

RUN go install -v github.com/ramya-rao-a/go-outline@latest \
    && go install -v github.com/cweill/gotests/gotests@latest \
    && go install -v github.com/fatih/gomodifytags@latest \
    && go install -v github.com/josharian/impl@latest \
    && go install -v github.com/haya14busa/goplay/cmd/goplay@latest \
    && go install -v honnef.co/go/tools/cmd/staticcheck@latest \
    && go install -v golang.org/x/tools/gopls@latest \
    && go install -v github.com/go-delve/delve/cmd/dlv@latest

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

RUN curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin v0.18.3