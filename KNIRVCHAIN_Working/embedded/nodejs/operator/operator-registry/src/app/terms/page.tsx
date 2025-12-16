// src/app/terms/page.tsx
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export default function TermsPage() {
  return (
    <div className="container mx-auto max-w-3xl py-12 px-4">
      <Card>
        <CardHeader>
          <CardTitle className="text-3xl font-bold">Terms of Service</CardTitle>
        </CardHeader>
        <CardContent className="space-y-6 text-foreground/80">
          <p>Welcome to the Operator Registry (AgentVerse)!</p>
          <p>These terms and conditions outline the rules and regulations for the use of the Operator Registry's Website, located at [Your Website URL].</p>
          <p>By accessing this website we assume you accept these terms and conditions. Do not continue to use the Operator Registry if you do not agree to take all of the terms and conditions stated on this page.</p>
          
          <h2 className="text-xl font-semibold text-foreground pt-4">1. Definitions</h2>
          <p>The following terminology applies to these Terms and Conditions, Privacy Statement and Disclaimer Notice and all Agreements: "Client", "You" and "Your" refers to you, the person log on this website and compliant to the Company’s terms and conditions. "The Company", "Ourselves", "We", "Our" and "Us", refers to our Company. "Party", "Parties", or "Us", refers to both the Client and ourselves. All terms refer to the offer, acceptance and consideration of payment necessary to undertake the process of our assistance to the Client in the most appropriate manner for the express purpose of meeting the Client’s needs in respect of provision of the Company’s stated services, in accordance with and subject to, prevailing law of Netherlands. Any use of the above terminology or other words in the singular, plural, capitalization and/or he/she or they, are taken as interchangeable and therefore as referring to same.</p>

          <h2 className="text-xl font-semibold text-foreground pt-4">2. Cookies</h2>
          <p>We employ the use of cookies. By accessing the Operator Registry, you agreed to use cookies in agreement with the Operator Registry's Privacy Policy.</p>
          <p>Most interactive websites use cookies to let us retrieve the user’s details for each visit. Cookies are used by our website to enable the functionality of certain areas to make it easier for people visiting our website. Some of our affiliate/advertising partners may also use cookies.</p>

          <h2 className="text-xl font-semibold text-foreground pt-4">3. License</h2>
          <p>Unless otherwise stated, the Operator Registry and/or its licensors own the intellectual property rights for all material on the Operator Registry. All intellectual property rights are reserved. You may access this from the Operator Registry for your own personal use subjected to restrictions set in these terms and conditions.</p>
          <p>You must not:</p>
          <ul className="list-disc pl-6 space-y-1">
            <li>Republish material from the Operator Registry</li>
            <li>Sell, rent or sub-license material from the Operator Registry</li>
            <li>Reproduce, duplicate or copy material from the Operator Registry</li>
            <li>Redistribute content from the Operator Registry</li>
          </ul>
          <p>This Agreement shall begin on the date hereof.</p>
          
          <p className="pt-4">...</p>
          <p><em>This is a placeholder Terms of Service page. Please replace this with your actual terms.</em></p>
        </CardContent>
      </Card>
    </div>
  );
}
