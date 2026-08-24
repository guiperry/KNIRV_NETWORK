import { FormEvent, useEffect, useState } from 'react';
import { useRole } from '../contexts/RoleContext';

type RiskClass = { id: string; display_name: string; description: string; domain: string; status: string };
type Submission = { id: string; status: string; validation_state?: string; decision_id?: string };

// Enterprise submissions stay same-origin: the API route forwards the session
// relay's bearer token to KNIRVSERVER and never exposes the backend address.
export default function SubmitVulnerability() {
  const { isAuthenticated, network } = useRole();
  const [classes, setClasses] = useState<RiskClass[]>([]);
  const [riskClass, setRiskClass] = useState('');
  const [organizationID, setOrganizationID] = useState('');
  const [organizationName, setOrganizationName] = useState('');
  const [submission, setSubmission] = useState<Submission | null>(null);
  const [error, setError] = useState('');

  const headers = () => ({ 'content-type': 'application/json', authorization: `Bearer ${localStorage.getItem('knirv_auth_token') || ''}`, 'x-knirv-network-id': network || 'local', 'idempotency-key': crypto.randomUUID() });
  useEffect(() => { fetch('/api/actuarial/risk-classes').then(r => r.ok ? r.json() : Promise.reject()).then((items: RiskClass[]) => setClasses(items.filter(item => item.domain === 'security_exploit' && item.status === 'active'))).catch(() => setError('Could not load available vulnerability classes.')); }, []);

  async function submit(event: FormEvent) {
    event.preventDefault(); setError('');
    try {
      const credential = await fetch('/api/actuarial/credentials/enterprise', { method: 'POST', headers: headers(), body: JSON.stringify({ organization_id: organizationID, organization_name: organizationName }) });
      if (!credential.ok) throw new Error((await credential.json()).error || 'Organization verification failed');
      const verified = await credential.json();
      const response = await fetch('/api/actuarial/submissions', { method: 'POST', headers: headers(), body: JSON.stringify({ enterprise_credential_id: verified.id, claimed_risk_class: riskClass, idempotency_key: crypto.randomUUID() }) });
      if (!response.ok) throw new Error((await response.json()).error || 'Submission creation failed');
      setSubmission(await response.json());
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Submission failed'); }
  }

  if (!isAuthenticated) return <main className="p-8"><h1>Submit a vulnerability</h1><p>Open this page from an authenticated KNIRVSERVER dashboard session.</p></main>;
  return <main className="mx-auto max-w-2xl p-8"><h1 className="text-3xl font-bold">Submit a vulnerability</h1><p className="mt-2 text-gray-600">Enterprise security-exploit submissions are verified against your dashboard session. Quotes and settlement are backend-derived.</p><form className="mt-8 space-y-4" onSubmit={submit}><label className="block">Organization ID<input required value={organizationID} onChange={e => setOrganizationID(e.target.value)} className="block w-full border p-2" /></label><label className="block">Organization name<input required value={organizationName} onChange={e => setOrganizationName(e.target.value)} className="block w-full border p-2" /></label><label className="block">Vulnerability class<select required value={riskClass} onChange={e => setRiskClass(e.target.value)} className="block w-full border p-2"><option value="">Select a class</option>{classes.map(item => <option key={item.id} value={item.id}>{item.display_name}</option>)}</select></label><button className="rounded bg-blue-700 px-4 py-2 text-white">Create protected submission</button></form>{error && <p className="mt-4 text-red-700">{error}</p>}{submission && <section className="mt-6 rounded border p-4"><h2 className="font-semibold">Submission created</h2><p>ID: {submission.id}</p><p>Status: {submission.status}</p><a className="mt-3 inline-block text-blue-700 underline" href={`/submissions/${encodeURIComponent(submission.id)}`}>Track validation and settlement</a></section>}</main>;
}
