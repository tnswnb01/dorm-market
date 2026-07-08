import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { AuthProvider } from './context/AuthContext'
import Navbar from './components/Navbar'
import ProtectedRoute from './components/ProtectedRoute'
import HomePage from './pages/HomePage'
import LoginPage from './pages/LoginPage'
import RegisterPage from './pages/RegisterPage'
import ListingDetailPage from './pages/ListingDetailPage'
import CreateListingPage from './pages/CreateListingPage'
import MyListingsPage from './pages/MyListingsPage'
import ConversationsPage from './pages/ConversationsPage'
import ChatPage from './pages/ChatPage'
import EditListingPage from './pages/EditListingPage'
import PurchaseHistoryPage from './pages/PurchaseHistoryPage'

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Navbar />
        <main className="max-w-container mx-auto px-4 py-6 pb-16 sm:px-6">
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
      </AuthProvider>
    </BrowserRouter>
  )
}
