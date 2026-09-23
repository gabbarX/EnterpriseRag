/**
 * Maps a rewind skip reason from the API to the notice shown to the user.
 *
 * An empty reason means the workspace was reset; the caller should not show
 * a skip notice in that case.
 */
export function rewindSkipMessage(reason: string): string {
  if (!reason) {
    return ''
  }
  switch (reason) {
    case 'NO_SANDBOX':
      return 'Conversation rewound; workspace was left unchanged (no sandbox is bound)'
    case 'NO_CHECKPOINT':
      return 'Conversation rewound; workspace was left unchanged (no checkpoint to restore)'
    default:
      return 'Conversation rewound; workspace was left unchanged'
  }
}
