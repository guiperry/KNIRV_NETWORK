import ResetPasswordClient from './reset-password-client';

type SearchParams = {
  access_token?: string;
  refresh_token?: string;
};

export default function ResetPasswordPage({
  searchParams,
}: {
  searchParams?: SearchParams;
}) {
  return <ResetPasswordClient searchParams={searchParams ?? {}} />;
}
