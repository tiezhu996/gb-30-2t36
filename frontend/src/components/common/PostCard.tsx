import { Card, Tag } from 'antd'
import { useNavigate } from 'react-router-dom'
import { useEffect } from 'react'
import type { CommunityPost } from '@/types/api'
import { formatDateTime } from '@/utils/dateFormat'
import LikeButton from './LikeButton'
import { usePostLikeStore } from '@/stores/postLikeStore'

export default function PostCard({ post }: { post: CommunityPost }) {
  const navigate = useNavigate()
  const seed = usePostLikeStore((s) => s.seed)

  useEffect(() => {
    seed([post])
  }, [post])

  return (
    <Card hoverable style={{ marginBottom: 12 }} onClick={() => navigate(`/community/${post.id}`)}>
      <Tag color={post.post_type === 'story' ? 'green' : 'orange'}>
        {post.post_type === 'story' ? '救助故事' : '寻主公告'}
      </Tag>
      <h3 style={{ margin: '8px 0' }}>{post.title}</h3>
      <p style={{ color: '#888' }}>{post.content?.slice(0, 80)}</p>
      <div style={{ color: '#aaa', fontSize: 12, display: 'flex', gap: 16, alignItems: 'center' }}>
        <span>{formatDateTime(post.created_at)}</span>
        <LikeButton postId={post.id} />
        <span>💬 {post.comment_count}</span>
      </div>
    </Card>
  )
}
