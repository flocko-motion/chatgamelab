import { createFileRoute } from "@tanstack/react-router";
import { EnterCodePage } from "@/features/auth";

/** Short link for invite and re-login codes: /code/<words>. */
export const Route = createFileRoute("/code/$code")({
  component: CodeLinkRoute,
});

function CodeLinkRoute() {
  const { code } = Route.useParams();
  return <EnterCodePage key={code} initialCode={code} />;
}
