import Head from "next/head";

export default function Custom500() {
  return (
    <>
      <Head>
        <title>KNIRV-SERVER - Internal Error</title>
        <meta name="robots" content="noindex" />
      </Head>
      <main
        style={{
          minHeight: "100vh",
          display: "grid",
          placeItems: "center",
          background: "linear-gradient(180deg, #000818 0%, #050b1a 100%)",
          color: "#e2e8f0",
          fontFamily: "system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
          padding: "2rem",
          textAlign: "center",
        }}
      >
        <section style={{ maxWidth: 560 }}>
          <p style={{ margin: 0, fontSize: 14, letterSpacing: "0.2em", textTransform: "uppercase", color: "#38bdf8" }}>
            KNIRV-SERVER
          </p>
          <h1 style={{ margin: "1rem 0 0.5rem", fontSize: "clamp(2.5rem, 8vw, 4.5rem)", lineHeight: 1 }}>
            Internal Server Error
          </h1>
          <p style={{ margin: 0, fontSize: 18, lineHeight: 1.6, color: "#cbd5e1" }}>
            The public wrapper could not complete the requested build or request.
            Please retry after the backend and export artifacts are available.
          </p>
        </section>
      </main>
    </>
  );
}
