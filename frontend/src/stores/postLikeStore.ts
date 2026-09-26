import { create } from 'zustand'
import { listLikedPostIds, likePost, unlikePost } from '@/api/post'

interface PostLikeState {
  // postId -> whether the current user has liked it
  likedMap: Record<number, boolean>
  // postId -> authoritative like count from the server
  countMap: Record<number, number>
  // posts with an in-flight request (double-click guard)
  pending: Record<number, boolean>
  loaded: boolean
  /** Loads the signed-in user's liked ids (used by cached home feed too). */
  loadLikedIds: () => Promise<void>
  /** Seeds ids/counts from a server-fetched post list. */
  seed: (posts: { id: number; like_count: number; liked?: boolean }[]) => void
  /** Adds the like (idempotent); double clicks collapse into one. */
  like: (postId: number) => Promise<boolean>
  /** Removes the like (idempotent); double clicks collapse into one. */
  unlike: (postId: number) => Promise<boolean>
  reset: () => void
}

export const usePostLikeStore = create<PostLikeState>((set, get) => ({
  likedMap: {},
  countMap: {},
  pending: {},
  loaded: false,

  loadLikedIds: async () => {
    const res = await listLikedPostIds()
    const likedMap: Record<number, boolean> = {}
    for (const id of res.post_ids) {
      likedMap[id] = true
    }
    set({ likedMap, loaded: true })
  },

  seed: (posts) => {
    set((state) => {
      const countMap = { ...state.countMap }
      const likedMap = { ...state.likedMap }
      for (const p of posts) {
        countMap[p.id] = p.like_count
        if (typeof p.liked === 'boolean') {
          likedMap[p.id] = p.liked
        }
      }
      return { countMap, likedMap }
    })
  },

  like: async (postId) => {
    // Same person clicking twice at the same moment only sends one request.
    if (get().pending[postId]) return false
    set((state) => ({ pending: { ...state.pending, [postId]: true } }))
    try {
      const res = await likePost(postId)
      set((state) => ({
        likedMap: { ...state.likedMap, [postId]: true },
        countMap: { ...state.countMap, [postId]: res.like_count },
      }))
      return res.liked
    } finally {
      set((state) => {
        const pending = { ...state.pending }
        delete pending[postId]
        return { pending }
      })
    }
  },

  unlike: async (postId) => {
    if (get().pending[postId]) return false
    set((state) => ({ pending: { ...state.pending, [postId]: true } }))
    try {
      const res = await unlikePost(postId)
      set((state) => ({
        likedMap: { ...state.likedMap, [postId]: false },
        countMap: { ...state.countMap, [postId]: res.like_count },
      }))
      return res.liked
    } finally {
      set((state) => {
        const pending = { ...state.pending }
        delete pending[postId]
        return { pending }
      })
    }
  },

  reset: () => set({ likedMap: {}, countMap: {}, pending: {}, loaded: false }),
}))
