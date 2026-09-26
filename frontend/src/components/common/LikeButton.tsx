import { message } from 'antd'
import { LikeFilled, LikeOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'
import { usePostLikeStore } from '@/stores/postLikeStore'

export default function LikeButton({
  postId,
  size,
  style,
}: {
  postId: number
  size?: number
  style?: React.CSSProperties
}) {
  const navigate = useNavigate()
  const { isLoggedIn } = useAuth()
  const liked = !!usePostLikeStore((s) => s.likedMap[postId])
  const count = usePostLikeStore((s) => s.countMap[postId]) ?? 0
  const pending = !!usePostLikeStore((s) => s.pending[postId])
  const like = usePostLikeStore((s) => s.like)
  const unlike = usePostLikeStore((s) => s.unlike)

  async function onClick(e: React.MouseEvent) {
    // Keep the card's navigation from firing.
    e.stopPropagation()
    if (pending) return
    if (!isLoggedIn) {
      message.warning('请先登录')
      navigate('/login')
      return
    }
    if (liked) {
      await unlike(postId)
      message.success('已取消点赞')
    } else {
      await like(postId)
      message.success('已点赞')
    }
  }

  return (
    <a
      onClick={onClick}
      aria-pressed={liked}
      aria-label={liked ? '取消点赞' : '点赞'}
      style={{
        cursor: pending ? 'wait' : 'pointer',
        color: liked ? '#2f6f4e' : 'inherit',
        fontWeight: liked ? 600 : 400,
        whiteSpace: 'nowrap',
        ...style,
      }}
    >
      {liked ? <LikeFilled style={{ fontSize: size }} /> : <LikeOutlined style={{ fontSize: size }} />}
      &nbsp;{count}
    </a>
  )
}
