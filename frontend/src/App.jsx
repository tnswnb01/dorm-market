import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { AuthProvider } from '@/features/auth/context/AuthContext'
import Navbar from '@/components/Navbar'
import Footer from '@/components/Footer'
import ProtectedRoute from '@/features/auth/components/ProtectedRoute'
import HomePage from '@/features/listings/pages/HomePage'
import LoginPage from '@/features/auth/pages/LoginPage'
import RegisterPage from '@/features/auth/pages/RegisterPage'
import ListingDetailPage from '@/features/listings/pages/ListingDetailPage'
import CreateListingPage from '@/features/listings/pages/CreateListingPage'
import MyListingsPage from '@/features/listings/pages/MyListingsPage'
import ConversationsPage from '@/features/chat/pages/ConversationsPage'
import ChatPage from '@/features/chat/pages/ChatPage'
import EditListingPage from '@/features/listings/pages/EditListingPage'
import PurchaseHistoryPage from '@/features/chat/pages/PurchaseHistoryPage'

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <div className="flex min-h-screen flex-col">
          <Navbar />
          <main className="mx-auto w-full max-w-container flex-1 px-4 py-6 pb-16 sm:px-6">
            <Routes>
              <Route path="/" element={<HomePage />} />
              <Route path="/login" element={<LoginPage />} />
              <Route path="/register" element={<RegisterPage />} />
              <Route path="/listings/:id" element={<ListingDetailPage />} />
              <Route
                path="/listings/new"
                element={
                  <ProtectedRoute>
                    <CreateListingPage />
                  </ProtectedRoute>
                }
              />
              <Route
                path="/my-listings"
                element={
                  <ProtectedRoute>
                    <MyListingsPage />
                  </ProtectedRoute>
                }
              />
              <Route
                path="/listings/:id/edit"
                element={
                  <ProtectedRoute>
                    <EditListingPage />
                  </ProtectedRoute>
                }
              />
              <Route
                path="/conversations"
                element={
                  <ProtectedRoute>
                    <ConversationsPage />
                  </ProtectedRoute>
                }
              />
              <Route
                path="/purchases"
                element={
                  <ProtectedRoute>
                    <PurchaseHistoryPage />
                  </ProtectedRoute>
                }
              />
              <Route
                path="/chat/:id"
                element={
                  <ProtectedRoute>
                    <ChatPage />
                  </ProtectedRoute>
                }
              />
            </Routes>
          </main>
          <Footer />
        </div>
      </AuthProvider>
    </BrowserRouter>
  )
}
