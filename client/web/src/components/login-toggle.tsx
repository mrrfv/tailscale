// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

import cx from "classnames"
import React, { useCallback, useMemo, useState } from "react"
import { ReactComponent as ChevronDown } from "src/assets/icons/chevron-down.svg"
import { ReactComponent as Eye } from "src/assets/icons/eye.svg"
import { ReactComponent as User } from "src/assets/icons/user.svg"
import { AuthResponse } from "src/hooks/auth"
import { useTSWebConnected } from "src/hooks/ts-web-connected"
import { NodeData } from "src/types"
import Button from "src/ui/button"
import Popover from "src/ui/popover"
import ProfilePic from "src/ui/profile-pic"
import { assertNever, isHTTPS } from "src/utils/util"

export default function LoginToggle({
  node,
  auth,
  newSession,
}: {
  node: NodeData
  auth: AuthResponse
  newSession: () => Promise<void>
}) {
  const [open, setOpen] = useState<boolean>(false)

  return (
    <Popover
      className="p-3 bg-white rounded-lg shadow flex flex-col max-w-[317px]"
      content={
        auth.serverMode === "readonly" ? (
          <ReadonlyModeContent auth={auth} />
        ) : auth.serverMode === "login" ? (
          <LoginModeContent auth={auth} node={node} />
        ) : auth.serverMode === "manage" ? (
          <ManageModeContent auth={auth} node={node} newSession={newSession} />
        ) : (
          assertNever(auth.serverMode)
        )
      }
      side="bottom"
      align="end"
      open={open}
      onOpenChange={setOpen}
      asChild
    >
      <div>
        {auth.authorized ? (
          <TriggerWhenManaging auth={auth} open={open} setOpen={setOpen} />
        ) : (
          <TriggerWhenReading auth={auth} open={open} setOpen={setOpen} />
        )}
      </div>
    </Popover>
  )
}

/**
 * TriggerWhenManaging is displayed as the trigger for the login popover
 * when the user has an active authorized managment session.
 */
function TriggerWhenManaging({
  auth,
  open,
  setOpen,
}: {
  auth: AuthResponse
  open: boolean
  setOpen: (next: boolean) => void
}) {
  return (
    <div
      className={cx(
        "w-[34px] h-[34px] p-1 rounded-full justify-center items-center inline-flex hover:bg-gray-300",
        {
          "bg-transparent": !open,
          "bg-gray-300": open,
        }
      )}
    >
      <button onClick={() => setOpen(!open)}>
        <ProfilePic size="medium" url={auth.viewerIdentity?.profilePicUrl} />
      </button>
    </div>
  )
}

/**
 * TriggerWhenReading is displayed as the trigger for the login popover
 * when the user is currently in read mode (doesn't have an authorized
 * management session).
 */
function TriggerWhenReading({
  auth,
  open,
  setOpen,
}: {
  auth: AuthResponse
  open: boolean
  setOpen: (next: boolean) => void
}) {
  return (
    <button
      className={cx(
        "pl-3 py-1 bg-gray-700 rounded-full flex justify-start items-center h-[34px]",
        { "pr-1": auth.viewerIdentity, "pr-3": !auth.viewerIdentity }
      )}
      onClick={() => setOpen(!open)}
    >
      <Eye />
      <div className="text-white leading-snug ml-2 mr-1">Viewing</div>
      <ChevronDown className="stroke-white w-[15px] h-[15px]" />
      {auth.viewerIdentity && (
        <ProfilePic
          className="ml-2"
          size="medium"
          url={auth.viewerIdentity.profilePicUrl}
        />
      )}
    </button>
  )
}

/**
 * PopoverContentHeader is the header for the login popover.
 */
function PopoverContentHeader({ auth }: { auth: AuthResponse }) {
  return (
    <div className="text-black text-sm font-medium leading-tight mb-1">
      {auth.authorized ? "Managing" : "Viewing"}
      {auth.viewerIdentity && ` as ${auth.viewerIdentity.loginName}`}
    </div>
  )
}

/**
 * PopoverContentFooter is the footer for the login popover.
 */
function PopoverContentFooter({ auth }: { auth: AuthResponse }) {
  return (
    auth.viewerIdentity && (
      <>
        <hr className="my-2" />
        <div className="flex items-center">
          <User className="flex-shrink-0" />
          <p className="text-gray-500 text-xs ml-2">
            We recognize you because you are accessing this page from{" "}
            <span className="font-medium">
              {auth.viewerIdentity.nodeName || auth.viewerIdentity.nodeIP}
            </span>
          </p>
        </div>
      </>
    )
  )
}

/**
 * ReadonlyModeContent is the body of the login popover when the web
 * client is being run in "readonly" server mode.
 */
function ReadonlyModeContent({ auth }: { auth: AuthResponse }) {
  return (
    <>
      <PopoverContentHeader auth={auth} />
      <p className="text-gray-500 text-xs">
        This web interface is running in read-only mode.{" "}
        <a
          href="https://tailscale.com/s/web-client-read-only"
          className="text-blue-700"
          target="_blank"
          rel="noreferrer"
        >
          Learn more &rarr;
        </a>
      </p>
      <PopoverContentFooter auth={auth} />
    </>
  )
}

/**
 * LoginModeContent is the body of the login popover when the web
 * client is being run in "login" server mode.
 */
function LoginModeContent({
  node,
  auth,
}: {
  node: NodeData
  auth: AuthResponse
}) {
  const { tsWebConnected, checkTSWebConnection } = useTSWebConnected(node.IPv4)
  const https = isHTTPS()
  // We can't run the ts web connection test when the webpage is loaded
  // over HTTPS. So in this case, we default to presenting a login button
  // with some helper text reminding the user to check their connection
  // themselves.
  const canConnect = https || tsWebConnected

  const handleLogin = useCallback(() => {
    // Must be connected over Tailscale to log in.
    // Send user to Tailscale IP and start check mode
    const manageURL = `http://${node.IPv4}:5252/?check=now`
    if (window.self !== window.top) {
      // If we're inside an iframe, open management client in new window.
      window.open(manageURL, "_blank")
    } else {
      window.location.href = manageURL
    }
  }, [node.IPv4])

  return (
    <div onMouseEnter={!canConnect ? checkTSWebConnection : undefined}>
      <PopoverContentHeader auth={auth} />
      {!canConnect ? (
        <>
          <p className="text-gray-500 text-xs">
            {!node.ACLAllowsAnyIncomingTraffic ? (
              // Tailnet ACLs don't allow access.
              <>
                The current tailnet policy file does not allow connecting to
                this device.
              </>
            ) : (
              // ACLs allow access, but user can't connect.
              <>
                Cannot access this device’s Tailscale IP. Make sure you are
                connected to your tailnet, and that your policy file allows
                access.
              </>
            )}{" "}
            <a
              href="https://tailscale.com/s/web-client-connection"
              className="text-blue-700"
              target="_blank"
              rel="noreferrer"
            >
              Learn more &rarr;
            </a>
          </p>
        </>
      ) : (
        // User can connect to Tailcale IP; sign in when ready.
        <>
          <p className="text-gray-500 text-xs">
            You can see most of this device’s details. To make changes, you need
            to sign in.
          </p>
          {https && (
            // we don't know if the user can connect over TS, so
            // provide extra tips in case they have trouble.
            <p className="text-gray-500 text-xs font-semibold pt-2">
              Make sure you are connected to your tailnet, and that your policy
              file allows access.
            </p>
          )}
          <SignInButton auth={auth} onClick={handleLogin} />
        </>
      )}
      <PopoverContentFooter auth={auth} />
    </div>
  )
}

/**
 * ManageModeContent is the body of the login popover when the web
 * client is being run in "manage" server mode.
 */
function ManageModeContent({
  auth,
  newSession,
}: {
  node: NodeData
  auth: AuthResponse
  newSession: () => void
}) {
  const handleLogin = useCallback(() => {
    if (window.self !== window.top) {
      // If we're inside an iframe, start session in new window.
      let url = new URL(window.location.href)
      url.searchParams.set("check", "now")
      window.open(url, "_blank")
    } else {
      newSession()
    }
  }, [newSession])

  const hasAnyPermissions = useMemo(
    () => Object.values(auth.viewerIdentity?.capabilities || {}).includes(true),
    [auth.viewerIdentity?.capabilities]
  )

  return (
    <>
      <PopoverContentHeader auth={auth} />
      {!auth.authorized &&
        (hasAnyPermissions ? (
          // User is connected over Tailscale, but needs to complete check mode.
          <>
            <p className="text-gray-500 text-xs">
              To make changes, sign in to confirm your identity. This extra step
              helps us keep your device secure.
            </p>
            <SignInButton auth={auth} onClick={handleLogin} />
          </>
        ) : (
          // User is connected over tailscale, but doesn't have permission to manage.
          <p className="text-gray-500 text-xs">
            You don’t have permission to make changes to this device, but you
            can view most of its details.{" "}
            <a
              href="https://tailscale.com/s/web-client-acls"
              className="text-blue-700"
              target="_blank"
              rel="noreferrer"
            >
              Learn more &rarr;
            </a>
          </p>
        ))}
      <PopoverContentFooter auth={auth} />
    </>
  )
}

function SignInButton({
  auth,
  onClick,
}: {
  auth: AuthResponse
  onClick: () => void
}) {
  return (
    <Button
      className={cx("text-center w-full mt-2", {
        "mb-2": auth.viewerIdentity,
      })}
      intent="primary"
      sizeVariant="small"
      onClick={onClick}
    >
      {auth.viewerIdentity ? "Sign in to confirm identity" : "Sign in"}
    </Button>
  )
}
