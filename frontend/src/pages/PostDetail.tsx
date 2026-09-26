import { Card, Tag, message } from 'antd'
import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { getPost, likePost } from '@/api/post'
import PostCommentList from '@/components/common/PostCommentList'
import type { CommunityPost } from '@/types/api'
import { formatDateTime } from '@/utils/dateFormat'
import { useAuth } from '@/hooks/useAuth'
import { useNavigate } from 'react-router-dom'

export default function PostDetail() {
  const { id } = useParams()
  const { isLoggedIn } = useAuth()
  const navigate = useNavigate()
  const [post, setPost] = useState<CommunityPost | null>(null)
  const [liking, setLiking] = useState(false)

  async function load() {
    setPost(await getPost(id!))
  }
  useEffect(() => {
    load()
  }, [id])

  async function like() {
    if (!isLoggedIn) {
      message.warning('请先登录')
      navigate('/login')
      return
    }
    if (liking) return
    setLiking(true)
    try {
      setPost(await likePost(post!.id))
    } finally {
      setLiking(false)
    }
  }

  if (!post) return <p>加载中…</p>
  return (
    <div style={{ maxWidth: 800, margin: '0 auto' }}>
      <Card>
        <Tag color={post.post_type === 'story' ? 'green' : 'orange'}>{post.post_type === 'story' ? '救助故事' : '寻主公告'}</Tag>
        <h1>{post.title}</h1>
        <p style={{ color: '#888' }}>{formatDateTime(post.created_at)}</p>
        <p style={{ lineHeight: 1.8 }}>{post.content}</p>
        <a
          onClick={like}
          style={{
            color: post.liked ? '#f5222d' : undefined,
            cursor: liking ? 'not-allowed' : 'pointer',
            opacity: liking ? 0.6 : 1,
          }}
        >
          {post.liked ? '❤️' : '👍'} {post.like_count}
          {post.liked ? ' 已点赞（再点一次取消）' : ' 点赞'}
        </a>
      </Card>
      <Card title="评论" style={{ marginTop: 16 }}>
        <PostCommentList postId={post.id} />
      </Card>
    </div>
  )
}
