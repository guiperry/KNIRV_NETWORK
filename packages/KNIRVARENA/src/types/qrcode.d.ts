declare module 'qrcode' { const QRCode: { toDataURL(value: string, options?: object): Promise<string> }; export default QRCode; }
