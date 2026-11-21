import { useEffect, useMemo, useRef, useState } from "react";
import { useDisclosure } from "@mantine/hooks";
import { useRecentlyRegisteredInstitutions } from "@/api/hooks";
import { markRegistrationsSeen, seenRegistrationIds } from "@/common/lib/seenRegistrations";

/**
 * The organisations that registered themselves in the last two weeks, for an
 * admin to check and suspend if something is wrong: how many this admin has not been told
 * about, and the notice that opens once per new one.
 */
export function useNewOrganisationsNotice() {
  const { data, isLoading } = useRecentlyRegisteredInstitutions();
  const [opened, { open, close }] = useDisclosure(false);
  const autoOpened = useRef(false);
  // Bumped when the notice closes, so the count follows at once.
  const [seenVersion, setSeenVersion] = useState(0);

  const organisations = useMemo(() => data ?? [], [data]);
  // eslint-disable-next-line react-hooks/exhaustive-deps -- re-read when the notice closes
  const seen = useMemo(() => seenRegistrationIds(), [seenVersion]);
  const unseen = organisations.filter((o) => !seen.has(o.id)).length;

  useEffect(() => {
    if (!isLoading && unseen > 0 && !autoOpened.current) {
      autoOpened.current = true;
      open();
    }
  }, [isLoading, unseen, open]);

  const dismiss = () => {
    markRegistrationsSeen(organisations.map((o) => o.id));
    setSeenVersion((v) => v + 1);
    close();
  };

  return { organisations, unseen, opened, open, dismiss };
}
