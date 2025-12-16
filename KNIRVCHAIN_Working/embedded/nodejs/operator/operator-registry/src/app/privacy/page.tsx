// src/app/privacy/page.tsx
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export default function PrivacyPage() {
  return (
    <div className="container mx-auto max-w-3xl py-12 px-4">
      <Card>
        <CardHeader>
          <CardTitle className="text-3xl font-bold">Privacy Policy</CardTitle>
        </CardHeader>
        <CardContent className="space-y-6 text-foreground/80">
          <p>Your privacy is important to us. It is the Operator Registry's policy to respect your privacy regarding any information we may collect from you across our website, [Your Website URL], and other sites we own and operate.</p>
          
          <h2 className="text-xl font-semibold text-foreground pt-4">1. Information We Collect</h2>
          <p>We only ask for personal information when we truly need it to provide a service to you. We collect it by fair and lawful means, with your knowledge and consent. We also let you know why we’re collecting it and how it will be used.</p>
          <p>Log data: When you visit our website, our servers may automatically log the standard data provided by your web browser. It may include your computer’s Internet Protocol (IP) address, your browser type and version, the pages you visit, the time and date of your visit, the time spent on each page, and other details.</p>
          <p>Device data: We may also collect data about the device you’re using to access our website. This data may include the device type, operating system, unique device identifiers, device settings, and geo-location data. What we collect can depend on the individual settings of your device and software. We recommend checking the policies of your device manufacturer or software provider to learn what information they make available to us.</p>
          <p>Personal information: We may ask for personal information, such as your name, email, social media profiles, phone/mobile number, home/mailing address, payment information.</p>

          <h2 className="text-xl font-semibold text-foreground pt-4">2. Legal Bases for Processing</h2>
          <p>We will process your personal information lawfully, fairly and in a transparent manner. We collect and process information about you only where we have legal bases for doing so.</p>
          <p>These legal bases depend on the services you use and how you use them, meaning we collect and use your information only where:</p>
          <ul className="list-disc pl-6 space-y-1">
            <li>It’s necessary for the performance of a contract to which you are a party or to take steps at your request before entering into such a contract (for example, when we provide a service you request from us);</li>
            <li>It satisfies a legitimate interest (which is not overridden by your data protection interests), such as for research and development, to market and promote our services, and to protect our legal rights and interests;</li>
            <li>You give us consent to do so for a specific purpose (for example, you might consent to us sending you our newsletter); or</li>
            <li>We need to process your data to comply with a legal obligation.</li>
          </ul>

          <h2 className="text-xl font-semibold text-foreground pt-4">3. Security of Your Personal Information</h2>
          <p>We retain collected information for as long as necessary to provide you with your requested service. What data we store, we’ll protect within commercially acceptable means to prevent loss and theft, as well as unauthorized access, disclosure, copying, use or modification.</p>
          <p>We don’t share any personally identifying information publicly or with third-parties, except when required to by law.</p>

          <p className="pt-4">...</p>
          <p><em>This is a placeholder Privacy Policy page. Please replace this with your actual policy.</em></p>
        </CardContent>
      </Card>
    </div>
  );
}
