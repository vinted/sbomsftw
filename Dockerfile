FROM ubuntu:20.04

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
RUN apt-get update && apt-get install rubygems build-essential git ruby-dev default-jre adoptopenjdk-8-hotspot \
    cmake pkg-config libssl-dev locales unzip clang nodejs cargo -y --no-install-recommends

# Fix up locales
RUN sed -i '/en_US.UTF-8/s/^# //g' /etc/locale.gen && locale-gen
ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US:en
ENV LC_ALL=en_US.UTF-8

# Install golang
WORKDIR /opt
RUN wget https://go.dev/dl/go1.17.7.linux-amd64.tar.gz \
  && tar -C /usr/local -xzf /opt/go1.17.7.linux-amd64.tar.gz && rm /opt/go1.17.7.linux-amd64.tar.gz
ENV PATH="/usr/local/go/bin:${PATH}"

# Install yarn & SBOM collection tools (cdxgen & cyclonedx-cli & cyclonedx-tools & licensee)
RUN npm install -g @appthreat/cdxgen @cyclonedx/bom yarn bower \
  && gem install cyclonedx-ruby licensee bundler bundler:1.9 bundler:1.17.3 \
  && wget https://github.com/CycloneDX/cyclonedx-cli/releases/download/v0.22.0/cyclonedx-linux-x64 \
  -O /usr/local/bin/cyclonedx-cli && chmod +x /usr/local/bin/cyclonedx-cli

# Build cyclonedx-cocoapods from source
RUN git clone https://github.com/CycloneDX/cyclonedx-cocoapods.git && cd cyclonedx-cocoapods \
  && gem install "$(gem build cyclonedx-cocoapods.gemspec | awk '{print $2}' | tail -n 1)" \
  && rm -rf /opt/cyclonedx-cocoapods

#Install Swift
RUN wget https://download.swift.org/swift-5.5.3-release/ubuntu2004/swift-5.5.3-RELEASE/swift-5.5.3-RELEASE-ubuntu20.04.tar.gz \
    && tar -xvzf swift-5.5.3-RELEASE-ubuntu20.04.tar.gz && mv swift-5.5.3-RELEASE-ubuntu20.04 /usr/local/bin/swift \
    && rm swift-5.5.3-RELEASE-ubuntu20.04.tar.gz && ldconfig /usr/swift/lib/python3
ENV PATH="/usr/local/bin/swift/usr/bin:${PATH}"

# Compile ORT with Java > 8. Other subsequent ORT runs must also use Java > 8
ENV JAVA_HOME="/usr/lib/jvm/default-java"
RUN git clone https://github.com/oss-review-toolkit/ort.git
RUN cd /opt/ort && ./gradlew installDist

# Disable root account and switch to a low-priv user
RUN groupadd -g 1000 satan && useradd -u 1000 -g satan -c "User for running the SBOM collector" -m satan \
	&& usermod --shell /usr/sbin/nologin root
USER satan:satan

WORKDIR /home/satan
ENV GEM_HOME="/home/satan/.gem"

#Install Android SDK & NDK
### Android environment variables
ENV ANDROID_HOME="/home/satan/android-sdk-linux"
ENV PATH="${PATH}:${ANDROID_HOME}/tools/bin:${ANDROID_HOME}/platform-tools:${ANDROID_HOME}/tools"
ENV ANDROID_NDK="/home/satan/android-ndk-linux"
ENV ANDROID_NDK_HOME="$ANDROID_NDK"
ENV ANDROID_NDK_VERSION="r23b"
ENV ANDROID_BUILD_TOOLS_VERSION="4333796"

### Setup Android SDK using Java 8 & Accept EULA
ENV JAVA_HOME="/usr/lib/jvm/adoptopenjdk-8-hotspot-amd64"
RUN  mkdir -p ~/.android && touch ~/.android/repositories.cfg && mkdir ~/android-sdk-linux && cd ~/android-sdk-linux \
    && wget https://dl.google.com/android/repository/sdk-tools-linux-${ANDROID_BUILD_TOOLS_VERSION}.zip \
    -q --output-document=sdk-tools.zip && unzip sdk-tools.zip && rm sdk-tools.zip \
    && echo y | sdkmanager "build-tools;28.0.2" "platforms;android-28" \
    && echo y | sdkmanager "extras;android;m2repository" "extras;google;m2repository" "extras;google;google_play_services" \
    && sdkmanager "cmake;3.6.4111459"

### Setup Android NDK
RUN wget https://dl.google.com/android/repository/android-ndk-${ANDROID_NDK_VERSION}-linux.zip \
    -q --output-document=android-ndk.zip && unzip android-ndk.zip && rm android-ndk.zip \
    && mv android-ndk-${ANDROID_NDK_VERSION} android-ndk-linux

# Install cyclonedx-gomod & cargo-cyclonedx for low-privilege user
RUN go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest && cargo install cargo-cyclonedx
ENV PATH="/home/satan/go/bin:${PATH}"