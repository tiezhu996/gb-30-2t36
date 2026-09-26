import { Card, Tag } from 'antd'
import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { getPost } from '@/api/post'
import PostCommentList from '@/components/common/PostCommentList'
import LikeButton from '@/components/common/LikeButton'
import type { CommunityPost } from '@/types/api'
import { formatDateTime } from '@/utils/dateFormat'
import { usePostLikeStore } from '@/stores/postLikeStore'

export default function PostDetail() {
  const { id } = useParams()
  const [post, setPost] = useState<CommunityPost | null>(null)
  const seed = usePostLikeStore((s) => s.seed)

  async function load() {
    const p = await getPost(id!)
    setPost(p)
    seed([p])
  }
  useEffect(() => {
    load()
  }, [id])

  if (!post) return <p>加载中…</p>
  return (
    <div style={{ maxWidth: 800, margin: '0 auto' }}>
      <Card>
        <Tag color={post.post_type === 'story' ? 'green' : 'orange'}>{post.post_type === 'story' ? '救助故事' : '寻主公告'}</Tag>
        <h1>{post.title}</h1>
        <p style={{ color: '#888' }}>{formatDateTime(post.created_at)}</p>
        <p style={{ lineHeight: 1.8 }}>{post.content}</p>
        <LikeButton postId={post.id} size={18} />
      </Card>
      <Card title="评论" style={{ marginTop: 16 }}>
        <PostCommentList postId={post.id} />
      </Card>
    </div>
  )
}
