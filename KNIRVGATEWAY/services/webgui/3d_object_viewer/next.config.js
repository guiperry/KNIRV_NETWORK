/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  webpack: (config, { isServer }) => {
    // Fixes npm packages that depend on `fs` module
    if (!isServer) {
      config.resolve.fallback = {
        ...config.resolve.fallback,
        fs: false,
        path: false,
        os: false,
        crypto: false,
        http: false,
        https: false,
        stream: false,
        zlib: false,
        util: false,
        url: false,
        net: false,
        tls: false,
        child_process: false,
        buffer: require.resolve('buffer/'),
      };
    }

    return config;
  },
  serverExternalPackages: ['three'],
  experimental: {
    
  },
};

module.exports = nextConfig;