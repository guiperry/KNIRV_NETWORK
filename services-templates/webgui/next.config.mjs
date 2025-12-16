/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  // Use Next.js's built-in Fast Refresh
  swcMinify: true,
  // Your other Next.js configurations can go here
  // For newer Next.js versions (e.g., 13.3+)
  output: 'export',

  // If you have images and want to use the default loader with `next export`,
  // you might need to configure a custom loader or ensure your images are
  // optimized at build time or served from a CDN.
  // For simplicity with local webview, you might disable image optimization
  // or ensure paths are relative.
  images: {
    unoptimized: true, // Simplest for local static export if image optimization is an issue
  },
  // If you have dynamic routes, ensure you have `generateStaticParams`
  // (or `getStaticPaths` in older versions) defined for them.
};

export default nextConfig;
