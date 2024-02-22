// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

import { useCallback, useEffect, useState } from "react"
import { isHTTPS } from "src/utils/util"

/**
 * useTSWebConnected hook is used to check whether the browser is able to
 * connect to the web client served at http://${nodeIPv4}:5252
 */
export function useTSWebConnected(nodeIPv4: string) {
  const [tsWebConnected, setTSWebConnected] = useState<boolean>(false)
  const [isRunningCheck, setIsRunningCheck] = useState<boolean>(false)

  const checkTSWebConnection = useCallback(() => {
    if (isHTTPS()) {
      // When page is loaded over HTTPS, the connectivity check will always
      // fail with a mixed-content error. In this case don't bother doing
      // the check.
      return
    }
    if (isRunningCheck) {
      return // already checking
    }
    setIsRunningCheck(true)
    fetch(`http://${nodeIPv4}:5252/ok`, { mode: "no-cors" })
      .then(() => {
        setTSWebConnected(true)
        setIsRunningCheck(false)
      })
      .catch(() => setIsRunningCheck(false))
  }, [isRunningCheck, nodeIPv4])

  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => checkTSWebConnection(), []) // checking connection for first time on page load

  return { tsWebConnected, checkTSWebConnection }
}
