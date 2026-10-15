import { createFileRoute } from "@tanstack/react-router";
import { EnterCodePage } from "@/features/auth";

export const Route = createFileRoute("/code/")({
  component: EnterCodePage,
});
