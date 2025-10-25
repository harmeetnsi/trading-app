
'use client';

import { useEffect, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { MessageCircle } from 'lucide-react';
import NavBar from '@/components/ui/NavBar';
import ChatMessage from '@/components/chat/ChatMessage';
import ChatInput from '@/components/chat/ChatInput';
import TypingIndicator from '@/components/chat/TypingIndicator';
import { api } from '@/lib/api';
import { useChatStore } from '@/lib/store';

export default function ChatPage() {
  const router = useRouter();
  const { messages, isTyping, setMessages, addMessage, setIsTyping } = useChatStore();
  const [loading, setLoading] = useState(true);
  const [ws, setWs] = useState<WebSocket | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const checkAuth = async () => {
      try {
        await api.getProfile();
        await loadMessages();
        connectWebSocket();
      } catch (error) {
        router.push('/auth/login');
      }
    };

    checkAuth();

    return () => {
      if (ws) {
        ws.close();
      }
    };
  }, []);

  useEffect(() => {
    scrollToBottom();
  }, [messages, isTyping]);

  const loadMessages = async () => {
    try {
      const response = await api.getMessages(50);
      if (response.success) {
        setMessages(response.data || []);
      }
    } catch (error) {
      console.error('Failed to load messages:', error);
    } finally {
      setLoading(false);
    }
  };

  const connectWebSocket = () => {
    const socket = api.createWebSocket();
    if (!socket) return;

    socket.onopen = () => {
      console.log('WebSocket connected');
    };

    socket.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);

        if (data.type === 'chat') {
          addMessage({
            id: data.data.id,
            role: data.data.role,
            content: data.content,
            created_at: data.data.created_at,
          });
        } else if (data.type === 'typing') {
          setIsTyping(data.data.is_typing);
        }
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error);
      }
    };

    socket.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    socket.onclose = () => {
      console.log('WebSocket disconnected');
      // Reconnect after 3 seconds
      setTimeout(() => {
        connectWebSocket();
      }, 3000);
    };

    setWs(socket);
  };

  const handleSendMessage = async (message: string, file?: File) => {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      alert('Chat is not connected. Please wait...');
      return;
    }

    let fileId: number | undefined;

    // Upload file if provided
    if (file) {
      try {
        const response = await api.uploadFile(file);
        if (response.success) {
          fileId = response.data.id;
        }
      } catch (error) {
        console.error('Failed to upload file:', error);
        alert('Failed to upload file');
        return;
      }
    }

    // Send message via WebSocket
    const messageData = {
      type: 'chat',
      content: message,
      file_id: fileId,
    };

    ws.send(JSON.stringify(messageData));
  };

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50">
        <NavBar />
        <div className="flex items-center justify-center h-[calc(100vh-64px)]">
          <div className="spinner border-primary-600"></div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 flex flex-col">
      <NavBar />

      <div className="flex-1 overflow-hidden flex flex-col max-w-4xl w-full mx-auto">
        {/* Chat header */}
        <div className="bg-white border-b border-gray-200 px-4 py-3">
          <h1 className="text-lg font-semibold text-gray-900">AI Trading Assistant</h1>
          <p className="text-sm text-gray-500">Ask me anything about trading, strategies, or upload files for analysis</p>
        </div>

        {/* Messages */}
        <div className="flex-1 overflow-y-auto px-4 py-6 bg-gray-50">
          {messages.length === 0 ? (
            <div className="text-center text-gray-500 mt-10">
              <MessageCircle className="w-16 h-16 mx-auto mb-4 text-gray-300" />
              <p className="text-lg font-medium">Start a conversation</p>
              <p className="text-sm mt-2">Ask questions or upload files for analysis</p>
            </div>
          ) : (
            <>
              {messages.map((msg) => (
                <ChatMessage
                  key={msg.id}
                  role={msg.role}
                  content={msg.content}
                  timestamp={msg.created_at}
                />
              ))}
              {isTyping && <TypingIndicator />}
            </>
          )}
          <div ref={messagesEndRef} />
        </div>

        {/* Input */}
        <ChatInput onSend={handleSendMessage} disabled={!ws || ws.readyState !== WebSocket.OPEN} />
      </div>
    </div>
  );
}
