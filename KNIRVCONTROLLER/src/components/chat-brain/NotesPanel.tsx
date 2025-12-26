// Notes Panel Component - Note-taking with Markdown Support
import React, { useState, useEffect } from 'react';
import { X, Save, Plus, FileText, Trash2, Eye } from 'lucide-react';
import { useChatBrain } from '../../contexts/ChatBrainContext';
import ReactMarkdown from 'react-markdown';

interface NotesPanelProps {
  onClose: () => void;
}

export function NotesPanel({ onClose }: NotesPanelProps) {
  const { notes, currentNote, setCurrentNote, saveNote, loadNotes, deleteNote } = useChatBrain();
  const [title, setTitle] = useState('');
  const [content, setContent] = useState('');
  const [isEditing, setIsEditing] = useState(false);
  const [showPreview, setShowPreview] = useState(false);

  useEffect(() => {
    loadNotes();
  }, [loadNotes]);

  useEffect(() => {
    if (currentNote) {
      setTitle(currentNote.title);
      setContent(currentNote.content);
      setIsEditing(true);
    } else {
      setTitle('');
      setContent('');
      setIsEditing(false);
    }
  }, [currentNote]);

  const handleSave = async () => {
    if (!title.trim() || !content.trim()) return;

    try {
      await saveNote(title, content);
      setTitle('');
      setContent('');
      setIsEditing(false);
      setShowPreview(false);
    } catch (error) {
      console.error('Failed to save note:', error);
    }
  };

  const handleNewNote = () => {
    setCurrentNote(null);
    setTitle('');
    setContent('');
    setIsEditing(true);
    setShowPreview(false);
  };

  const handleSelectNote = (note: any) => {
    setCurrentNote(note);
  };

  const handleDelete = async (noteId: string, e: React.MouseEvent) => {
    e.stopPropagation();

    if (!confirm('Are you sure you want to delete this note?')) return;

    try {
      await deleteNote(noteId);
    } catch (error) {
      console.error('Failed to delete note:', error);
    }
  };

  return (
    <div className="fixed right-0 top-0 bottom-0 w-[600px] bg-gray-800 border-l border-gray-700 z-50 flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-gray-700 bg-gray-800/50">
        <h2 className="text-lg font-semibold text-white">Notes</h2>
        <div className="flex items-center space-x-2">
          {isEditing && (
            <button
              onClick={() => setShowPreview(!showPreview)}
              className="text-gray-400 hover:text-white transition-colors p-2 rounded hover:bg-gray-700"
              title={showPreview ? 'Edit' : 'Preview'}
            >
              <Eye size={18} />
            </button>
          )}
          <button
            onClick={handleNewNote}
            className="text-gray-400 hover:text-white transition-colors p-2 rounded hover:bg-gray-700"
            title="New note"
          >
            <Plus size={18} />
          </button>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-white transition-colors p-2 rounded hover:bg-gray-700"
            title="Close"
          >
            <X size={20} />
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 flex overflow-hidden">
        {/* Notes List */}
        {!isEditing && (
          <div className="flex-1 overflow-y-auto p-4 space-y-2">
            {notes.length === 0 ? (
              <div className="flex flex-col items-center justify-center h-full text-gray-400">
                <FileText size={48} className="mb-4 opacity-50" />
                <p className="text-lg mb-2">No notes yet</p>
                <button
                  onClick={handleNewNote}
                  className="text-sm bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg transition-colors"
                >
                  Create your first note
                </button>
              </div>
            ) : (
              notes.map((note) => (
                <div
                  key={note.id}
                  onClick={() => handleSelectNote(note)}
                  className="bg-gray-900 border border-gray-700 rounded-lg p-3 cursor-pointer hover:border-blue-500 transition-colors"
                >
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <h3 className="text-white font-medium mb-1">{note.title}</h3>
                      <p className="text-sm text-gray-400 line-clamp-2">
                        {note.content.substring(0, 100)}
                        {note.content.length > 100 ? '...' : ''}
                      </p>
                      <p className="text-xs text-gray-500 mt-2">
                        {new Date(note.timestamp).toLocaleDateString()}
                      </p>
                    </div>
                    <button
                      onClick={(e) => handleDelete(note.id, e)}
                      className="text-gray-400 hover:text-red-400 transition-colors ml-2"
                      title="Delete note"
                    >
                      <Trash2 size={16} />
                    </button>
                  </div>
                </div>
              ))
            )}
          </div>
        )}

        {/* Editor/Preview */}
        {isEditing && (
          <div className="flex-1 flex flex-col overflow-hidden">
            {/* Title Input */}
            <div className="p-4 border-b border-gray-700">
              <input
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Note title..."
                className="w-full bg-gray-900 text-white text-xl font-semibold border-none focus:outline-none placeholder-gray-500"
              />
            </div>

            {/* Content Editor/Preview */}
            <div className="flex-1 overflow-y-auto p-4">
              {showPreview ? (
                <div className="prose prose-invert max-w-none">
                  <ReactMarkdown>{content}</ReactMarkdown>
                </div>
              ) : (
                <textarea
                  value={content}
                  onChange={(e) => setContent(e.target.value)}
                  placeholder="Write your note in Markdown..."
                  className="w-full h-full bg-transparent text-white border-none focus:outline-none resize-none placeholder-gray-500"
                />
              )}
            </div>

            {/* Actions */}
            <div className="p-4 border-t border-gray-700 flex items-center justify-between">
              <button
                onClick={() => {
                  setIsEditing(false);
                  setCurrentNote(null);
                }}
                className="text-gray-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleSave}
                disabled={!title.trim() || !content.trim()}
                className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg
                           disabled:opacity-50 disabled:cursor-not-allowed transition-colors
                           flex items-center space-x-2"
              >
                <Save size={16} />
                <span>Save Note</span>
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
