# InfraForge - 专业 HPC 性能测试镜像
FROM nvcr.io/nvidia/cuda:12.8.1-devel-ubuntu24.04 as builder

# 版本定义 - 方便维护和升级
ARG NCCL_TESTS_VERSION=2.17.6
ARG OSU_VERSION=7.5.1
ARG IOR_VERSION=4.0.0
ARG PERFTEST_VERSION=25.07.0-0.104
ARG FABTESTS_VERSION=2.3.0
ARG SOCKPERF_VERSION=3.10
ARG EFA_INSTALLER_VERSION=1.44.0
ARG AWS_OFI_NCCL_VERSION=1.17.2
ARG GDRCOPY_VERSION=2.5.1
ARG NCCL_VERSION=2.28.9-1
ARG NVBANDWIDTH_VERSION=0.8
ARG NVSHMEM_VERSION=3.4.5
ARG CUDA_HOME=/usr/local/cuda

# 系统更新和清理预装包
RUN apt-get update -y && apt-get upgrade -y \
    && apt-get remove -y --allow-change-held-packages \
        ibverbs-utils libibverbs-dev libibverbs1 libmlx5-1 libnccl2 libnccl-dev \
    && rm -rf /opt/hpcx /usr/local/mpi \
    && rm -f /etc/ld.so.conf.d/hpcx.conf \
    && ldconfig \
    && apt-get purge -y cuda-compat-* \
    && rm -rf /var/lib/apt/lists/*

ENV OPAL_PREFIX=

# 安装所有构建依赖和 EFA 运行时依赖
RUN DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    apt-utils autoconf automake build-essential check cmake curl git gcc gdb \
    kmod libsubunit-dev libtool libhwloc-dev libpci-dev pkg-config sudo \
    python3-dev python3-setuptools python3.12-venv unzip libboost-program-options-dev \
    pciutils tcl libnl-3-200 libnl-3-dev libnl-route-3-200 libnl-route-3-dev \
    udev dmidecode ethtool iproute2 libevent-core-2.1-7t64 libevent-pthreads-2.1-7t64 \
    libevent-dev environment-modules \
    && . /etc/profile \
    && rm -rf /var/lib/apt/lists/*

# 安装 AWS CLI v2
RUN curl "https://awscli.amazonaws.com/awscli-exe-linux-$(uname -m).zip" -o "awscliv2.zip" \
    && unzip awscliv2.zip && ./aws/install || true \
    && rm -rf awscliv2.zip aws

# 构建所有组件 (合并 RUN 减少层数)
RUN set -e \
    # GDRCopy
    && curl -JLOk https://github.com/NVIDIA/gdrcopy/archive/refs/tags/v${GDRCOPY_VERSION}.tar.gz \
    && tar xf gdrcopy-${GDRCOPY_VERSION}.tar.gz \
    && cd gdrcopy-${GDRCOPY_VERSION} && make prefix=/infraforge install && cd .. \
    && rm -rf gdrcopy-* \
    \
    # EFA Installer
    && curl -O https://efa-installer.amazonaws.com/aws-efa-installer-${EFA_INSTALLER_VERSION}.tar.gz \
    && tar xf aws-efa-installer-${EFA_INSTALLER_VERSION}.tar.gz \
    && cd aws-efa-installer && ./efa_installer.sh -y -g -d --skip-kmod --skip-limit-conf --no-verify && cd .. \
    && rm -rf aws-efa-installer* \
    \
    # NCCL
    && curl -JLOk https://github.com/NVIDIA/nccl/archive/refs/tags/v${NCCL_VERSION}.tar.gz \
    && tar xf nccl-${NCCL_VERSION}.tar.gz \
    && cd nccl-${NCCL_VERSION} \
    && make -j $(nproc) src.build CUDA_HOME=/usr/local/cuda PREFIX=/infraforge \
       NVCC_GENCODE="-gencode=arch=compute_70,code=sm_70 -gencode=arch=compute_75,code=sm_75 -gencode=arch=compute_80,code=sm_80 -gencode=arch=compute_86,code=sm_86 -gencode=arch=compute_89,code=sm_89 -gencode=arch=compute_90,code=sm_90 -gencode=arch=compute_100,code=sm_100" \
    && make PREFIX=/infraforge install && cd .. \
    && rm -rf nccl-* \
    \
    # NVSHMEM
    && curl -L https://developer.download.nvidia.com/compute/redist/nvshmem/${NVSHMEM_VERSION}/source/nvshmem_src_cuda12-all-all-${NVSHMEM_VERSION}.tar.gz -o nvshmem_src.txz \
    && tar -xf nvshmem_src.txz \
    && cd nvshmem_src && mkdir -p build && cd build \
    && cmake -DNVSHMEM_PREFIX=/infraforge -DCMAKE_INSTALL_PREFIX=/infraforge \
       -DCUDA_HOME=/usr/local/cuda \
       -DCMAKE_BUILD_TYPE=Release \
       -DCMAKE_CUDA_ARCHITECTURES="75;80;86;89;90;100" \
       -DCMAKE_CUDA_FLAGS="-Wno-deprecated-gpu-targets" \
       -DNVSHMEM_USE_GDRCOPY=1 -DGDRCOPY_HOME=/infraforge \
       -DNVSHMEM_USE_NCCL=1 -DNCCL_HOME=/infraforge \
       -DNVSHMEM_LIBFABRIC_SUPPORT=1 -DLIBFABRIC_HOME=/opt/amazon/efa \
       -DNVSHMEM_MPI_SUPPORT=1 -DMPI_HOME=/opt/amazon/openmpi \
       -DNVSHMEM_PMIX_SUPPORT=1 -DPMIX_HOME=/opt/amazon/pmix \
       -DNVSHMEM_BUILD_TESTS=1 -DNVSHMEM_BUILD_EXAMPLES=1 \
       -DNVSHMEM_IBRC_SUPPORT=1 -DNVSHMEM_IBGDA_SUPPORT=1 \
       -DNVSHMEM_DEBUG=WARN -DNVSHMEM_TRACE=1 .. \
    && make -j $(nproc) && make install && cd ../.. \
    && rm -rf nvshmem_src* \
    \
    # 创建 NVSHMEM 工具的便捷链接
    && ln -sf /infraforge/bin/perftest/device/pt-to-pt/* /infraforge/bin/ \
    && ln -sf /infraforge/bin/perftest/device/coll/* /infraforge/bin/ \
    && ln -sf /infraforge/bin/perftest/device/tile/* /infraforge/bin/ \
    && ln -sf /infraforge/bin/perftest/host/pt-to-pt/* /infraforge/bin/ \
    && ln -sf /infraforge/bin/perftest/host/coll/* /infraforge/bin/ \
    && ln -sf /infraforge/bin/perftest/host/init/* /infraforge/bin/

# 构建网络和 MPI 工具
RUN set -e \
    # AWS-OFI-NCCL
    && curl -OL https://github.com/aws/aws-ofi-nccl/releases/download/v${AWS_OFI_NCCL_VERSION}/aws-ofi-nccl-${AWS_OFI_NCCL_VERSION}.tar.gz \
    && tar xf aws-ofi-nccl-${AWS_OFI_NCCL_VERSION}.tar.gz \
    && cd aws-ofi-nccl-${AWS_OFI_NCCL_VERSION} \
    && ./configure --prefix=/infraforge --with-mpi=/opt/amazon/openmpi --with-libfabric=/opt/amazon/efa \
       --with-cuda=/usr/local/cuda --with-nccl=/infraforge --enable-platform-aws \
    && make -j $(nproc) && make install && cd .. \
    && rm -rf aws-ofi-nccl-* \
    \
    # NCCL Tests
    && curl -JLOk https://github.com/NVIDIA/nccl-tests/archive/refs/tags/v${NCCL_TESTS_VERSION}.tar.gz \
    && tar xf nccl-tests-${NCCL_TESTS_VERSION}.tar.gz \
    && cd nccl-tests-${NCCL_TESTS_VERSION} \
    && make -j $(nproc) MPI=1 MPI_HOME=/opt/amazon/openmpi CUDA_HOME=/usr/local/cuda NCCL_HOME=/infraforge \
       NVCC_GENCODE="-gencode=arch=compute_70,code=sm_70 -gencode=arch=compute_75,code=sm_75 -gencode=arch=compute_80,code=sm_80 -gencode=arch=compute_86,code=sm_86 -gencode=arch=compute_89,code=sm_89 -gencode=arch=compute_90,code=sm_90 -gencode=arch=compute_100,code=sm_100" \
    && cp build/*_perf /infraforge/bin/ && cd .. \
    && rm -rf nccl-tests-* \
    \
    # nvbandwidth - GPU 内存带宽测试工具 (支持多节点)
    && curl -JLOk https://github.com/NVIDIA/nvbandwidth/archive/refs/tags/v${NVBANDWIDTH_VERSION}.tar.gz \
    && tar xf nvbandwidth-${NVBANDWIDTH_VERSION}.tar.gz \
    && cd nvbandwidth-${NVBANDWIDTH_VERSION} \
    && PATH=/opt/amazon/openmpi/bin:$PATH cmake -DMULTINODE=1 . \
    && make -j $(nproc) \
    && cp nvbandwidth /infraforge/bin/ && cd .. \
    && rm -rf nvbandwidth-* \
    \
    # OSU Benchmarks with NCCL support
    && curl -O https://mvapich.cse.ohio-state.edu/download/mvapich/osu-micro-benchmarks-${OSU_VERSION}.tar.gz \
    && tar xf osu-micro-benchmarks-${OSU_VERSION}.tar.gz \
    && cd osu-micro-benchmarks-${OSU_VERSION} \
    && ./configure --prefix=/infraforge CC=/opt/amazon/openmpi/bin/mpicc CXX=/opt/amazon/openmpi/bin/mpicxx \
       --enable-cuda --with-cuda=/usr/local/cuda \
       --enable-ncclomb --with-nccl=/infraforge \
    && make -j $(nproc) && make install \
    && ln -sf /infraforge/libexec/osu-micro-benchmarks/mpi/pt2pt/* /infraforge/bin/ \
    && ln -sf /infraforge/libexec/osu-micro-benchmarks/mpi/collective/* /infraforge/bin/ \
    && ln -sf /infraforge/libexec/osu-micro-benchmarks/mpi/one-sided/* /infraforge/bin/ \
    && ln -sf /infraforge/libexec/osu-micro-benchmarks/mpi/congestion/* /infraforge/bin/ \
    && ln -sf /infraforge/libexec/osu-micro-benchmarks/xccl/pt2pt/* /infraforge/bin/ \
    && ln -sf /infraforge/libexec/osu-micro-benchmarks/xccl/collective/* /infraforge/bin/ \
    && cd .. && rm -rf osu-micro-benchmarks-*

# 构建 I/O 和网络测试工具
RUN set -e \
    # IOR
    && curl -JLOk https://github.com/hpc/ior/archive/refs/tags/${IOR_VERSION}.tar.gz \
    && tar xf ior-${IOR_VERSION}.tar.gz \
    && cd ior-${IOR_VERSION} && ./bootstrap \
    && ./configure --prefix=/infraforge --with-mpiio CC=/opt/amazon/openmpi/bin/mpicc \
    && make -j $(nproc) && make install && cd .. \
    && rm -rf ior-* \
    \
    # perftest
    && curl -JLOk https://github.com/linux-rdma/perftest/archive/refs/tags/${PERFTEST_VERSION}.tar.gz \
    && tar xf perftest-${PERFTEST_VERSION}.tar.gz \
    && cd perftest-${PERFTEST_VERSION} && ./autogen.sh \
    && ./configure --prefix=/infraforge && make -j $(nproc) && make install && cd .. \
    && rm -rf perftest-* \
    \
    # fabtests
    && curl -JLOk https://github.com/ofiwg/libfabric/releases/download/v${FABTESTS_VERSION}/fabtests-${FABTESTS_VERSION}.tar.bz2 \
    && tar xf fabtests-${FABTESTS_VERSION}.tar.bz2 \
    && cd fabtests-${FABTESTS_VERSION} \
    && ./configure --prefix=/infraforge --with-libfabric=/opt/amazon/efa \
    && make -j $(nproc) && make install && cd .. \
    && rm -rf fabtests-* \
    \
    # sockperf
    && curl -JLOk https://github.com/Mellanox/sockperf/archive/refs/tags/${SOCKPERF_VERSION}.tar.gz \
    && tar xf sockperf-${SOCKPERF_VERSION}.tar.gz \
    && cd sockperf-${SOCKPERF_VERSION} && ./autogen.sh \
    && ./configure --prefix=/infraforge && make -j $(nproc) && make install && cd .. \
    && rm -rf sockperf-* \
    \
    # 最终清理
    && apt-get autoremove -y && apt-get clean \
    && rm -f /infraforge/License.txt /infraforge/changelog /infraforge/git_commit.txt /infraforge/version.txt \
    && find /infraforge -type f -executable ! -type l -exec strip {} \; \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# 运行时镜像 - 只保留必要文件
FROM nvcr.io/nvidia/cuda:12.8.1-runtime-ubuntu24.04
ARG EFA_INSTALLER_VERSION=1.44.0

# 安装运行时依赖
RUN DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    iperf3 curl python3 python3.12-venv wget git fio stress-ng netcat-openbsd nvtop \
    pciutils tcl libnl-3-200 libnl-route-3-200 udev dmidecode ethtool iproute2 \
    libevent-core-2.1-7t64 libevent-pthreads-2.1-7t64 libhwloc15 unzip apt-utils \
    && mkdir -p /infraforge/bin \
    && python3 -m venv /infraforge/python/venv \
    && /infraforge/python/venv/bin/pip install --upgrade pip setuptools wheel \
    && ln -sf /infraforge/python/venv/bin/python /infraforge/bin/ \
    && ln -sf /infraforge/python/venv/bin/pip /infraforge/bin/ \
    && apt-get remove -y --allow-change-held-packages libnccl2 || true \
    && apt-get autoremove -y

# 安装 AWS CLI v2
RUN curl "https://awscli.amazonaws.com/awscli-exe-linux-$(uname -m).zip" -o "awscliv2.zip" \
    && unzip awscliv2.zip && ./aws/install || true \
    && rm -rf awscliv2.zip aws

# 构建所有组件 (合并 RUN 减少层数)
# EFA Installer, kubectl & helm
RUN set -e \
    && curl -O https://efa-installer.amazonaws.com/aws-efa-installer-${EFA_INSTALLER_VERSION}.tar.gz \
    && tar zxf aws-efa-installer-${EFA_INSTALLER_VERSION}.tar.gz \
    && cd aws-efa-installer && ./efa_installer.sh -y -g -d --skip-kmod --skip-limit-conf --no-verify && cd .. \
    && rm -rf aws-efa-installer* \
    && mkdir -p /opt/kube \
    && curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/$(dpkg --print-architecture)/kubectl" \
    && chmod +x kubectl && mv kubectl /opt/kube/kubectl \
    && ln -sf /opt/kube/kubectl /usr/local/bin/kubectl \
    && curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash \
    && if [ -d "/usr/local/cuda/lib64" ]; then \
           cd /usr/local/cuda/lib64 && \
           for major in $(ls *.so.[0-9]* 2>/dev/null | grep -E '\.so\.[0-9]+$' | sort -u); do \
               base=$(echo $major | sed 's/\.so\.[0-9]*$/.so/'); \
               ln -snf $major $base; \
           done; \
       fi \
    && rm -rf /var/lib/apt/lists/*

# 复制必要的运行时组件
#COPY --from=builder /opt/amazon /opt/amazon
COPY --from=builder /infraforge /infraforge

# 设置环境变量
ENV LD_LIBRARY_PATH=/usr/local/cuda/extras/CUPTI/lib64:/opt/amazon/openmpi/lib:/infraforge/lib:/opt/amazon/efa/lib:/usr/local/lib:$LD_LIBRARY_PATH \
    PATH=/infraforge/bin:/opt/amazon/openmpi/bin:/opt/amazon/efa/bin:/opt/kube:$PATH \
    LIBRARY_PATH=/infraforge/lib:$LIBRARY_PATH \
    CPATH=/infraforge/include:$CPATH \
    CUDA_HOME=/usr/local/cuda \
    FI_EFA_USE_DEVICE_RDMA=0 \
    FI_EFA_ENABLE_SHM_TRANSFER=0 \
    OMPI_ALLOW_RUN_AS_ROOT=1 \
    OMPI_ALLOW_RUN_AS_ROOT_CONFIRM=1 \
    OMPI_MCA_plm_rsh_no_tree_spawn=1 \
    OMPI_MCA_mtl=ofi \
    OMPI_MCA_pml=cm \
    OMPI_MCA_mtl_ofi_provider_exclude=shm,sockets,udp,rstream,usnic,net \
    OPAL_PREFIX=/opt/amazon/openmpi \
    NCCL_SOCKET_IFNAME=^docker,lo,veth \
    NCCL_NET_PLUGIN=/infraforge/lib/libnccl-net.so \
    NCCL_TUNER_PLUGIN=/infraforge/lib/libnccl-ofi-tuner.so \
    NCCL_CROSS_NIC=1 \
    PMIX_MCA_gds=hash \
    NCCL_DEBUG=INFO \
    NVSHMEM_HOME=/infraforge \
    NVSHMEM_REMOTE_TRANSPORT=libfabric \
    NVSHMEM_LIBFABRIC_PROVIDER=efa

# 创建测试脚本
RUN echo '#!/bin/bash\n\
echo "=== InfraForge HPC Performance Test Suite ==="\n\
echo "NCCL Tests: all_reduce_perf, all_gather_perf, etc."\n\
echo "NVSHMEM: nvshmem_test, nvshmem_perf_test, etc."\n\
echo "OSU Benchmarks: osu_latency, osu_bw, osu_allreduce, etc."\n\
echo "IOR: ior (I/O performance)"\n\
echo "fio: Flexible I/O tester"\n\
echo "stress-ng: System stress testing"\n\
echo "perftest: ib_send_bw, ib_send_lat, etc."\n\
echo "fabtests: fi_pingpong, fi_rdm_pingpong, etc."\n\
echo "sockperf: Network latency and throughput testing"\n\
echo "iperf3: TCP performance"\n\
echo "netcat: Network connectivity testing"\n\
echo "kubectl: Kubernetes cluster management"\n\
echo "helm: Kubernetes package manager"\n\
echo "GDRCopy: gdrcopy_sanity, gdrcopy_copybw"\n\
echo "AWS-OFI-NCCL: Enhanced EFA performance"\n\
echo "nvbandwidth: NVIDIA GPU memory bandwidth testing"\n\
echo "AWS CLI v2: Amazon Web Services command line interface"\n\
echo "All components unified under /infraforge"\n\
echo "Architecture: $(uname -m)"\n\
touch /tmp/compilation_done\n\
if [ $# -eq 0 ]; then\n\
  sleep 365d\n\
else\n\
  exec "$@"\n\
fi\n\
' > /infraforge/test-suite.sh && chmod +x /infraforge/test-suite.sh

WORKDIR /infraforge
ENTRYPOINT ["/infraforge/test-suite.sh"]
CMD ["/bin/bash"]
